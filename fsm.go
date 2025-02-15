package main

import (
	"context"
	"runtime/debug"
	"time"
)

type globalConnEvent int

const (
	INCOMING globalConnEvent = iota
	EMPTY
)

// luckily I use int for them all
var (
	state MCState = STOPPED
	// we don't need mutex for cnt because with channel it's guaranteed to be processed sequently
	globalConnEventChan = make(chan globalConnEvent)
	// the channel used to pass channel to build directional communication
)
var cnt int

func InitState() {
	go handleCnt()
	go handleState()
}
func handleCnt() { //goroutine only, handles cnt related message
	for {
		cntEvent := <-CntChan
		switch cntEvent {
		case INCREASE:
			GetLogger().Debug("handleCnt(): connection cnt increased")
			cnt++
			if cnt == 1 { //which means it is 0->1
				globalConnEventChan <- INCOMING
			}
		case DECREASE:
			GetLogger().Debug("handleCnt(): connection cnt decreased")
			cnt--
			if cnt < 0 {
				debug.Stack()
				panic("cnt processor was written wrong!")
			} else if cnt == 0 {
				globalConnEventChan <- EMPTY
			}
		}
	}
}

// STOPPED->BOOTING->RUNNING<->WAITING->STOPPING->STOPPED
var waitChan chan struct{}

// no we don't need cmdChan, we just make it work immediately
func handleWaitingToRunning() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[RUNNING])
	waitChan <- struct{}{}
	_, ok := <-waitChan
	if ok {
		state = RUNNING //happens immediately
	} //else it's too late
}
func handleRunningToWaiting() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[WAITING])
	waitChan = make(chan struct{})
	go waitingThread()
	state = WAITING
}

// TODO: work with daemon
func handleWaitingToStopping() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[STOPPING])
	go stoppingThread()
	state = STOPPING
}
func stoppingThread() {
	DaemonChanTX <- struct{}{}
	<-DaemonChanRX
	<-DaemonChanRX
	ConnSignalChan <- STOPPING
	handleStoppingToStopped()
}
func handleStoppingToStopped() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[STOPPED])
	state = STOPPED
}
func bootingThread() {
	DaemonChanTX <- struct{}{}
	<-DaemonChanRX
	<-DaemonChanRX
	handleBootingToRunning()
}

var runningChan chan struct{}

func handleBootingToRunning() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[RUNNING])
	state = RUNNING
	ConnSignalChan <- RUNNING
}
func handleStoppedToBooting() {
	GetLogger().Infof("Server is currently at %s state", stateToStr[BOOTING])
	state = BOOTING
	bootingThread()
}
func waitingThread() {
	defer close(waitChan)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Minute)
	defer cancel()
	select {
	case <-ctx.Done():
		handleWaitingToStopping()
		return
	case <-waitChan:
		waitChan <- struct{}{}
		return
	}
}
func handleState() { //goroutine only, handles both write and read
	logger.Debug("handleState(): ready")
	multiChan := NewDynamicMultiChan[MCState](true, 1)
	for {
		select {
		case nchan, _ := <-QueryChanChan:
			multiChan.Add(nchan)
		case event, _ := <-globalConnEventChan:
			// wait until transformation finishes
			switch event {
			case INCOMING:
				switch state {
				case STOPPED:
					handleStoppedToBooting()
				// case STOPPING: // it shouldn't be there, too, should be handled with conn.go
				// case BOOTING: //shouldn't be there too
				case WAITING:
					handleWaitingToRunning()
				//case RUNNING:
				default:
					panic("handleState():written wrong!" + stateToStr[state])
				}
			case EMPTY:
				switch state {
				case RUNNING:
					handleRunningToWaiting()
				//case BOOTING:
				default:
					panic("handleState():written wrong!" + stateToStr[state])
				}
			}
		case <-multiChan.RX:
			multiChan.TX <- state
		}
	}

}
