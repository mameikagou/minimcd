package main

import (
	"context"
	"reflect"
	"runtime/debug"
	"time"
)

type globalConnEvent int

const (
	INCOMING globalConnEvent = iota
	EMPTY
)

// luckily I use int for them all
type GenericChan chan int
type QueryChan chan MCState

var (
	state MCState = STOPPED
	// we don't need mutex for cnt because with channel it's guaranteed to be processed sequently
	CntChan   = make(chan CntEvent)
	eventChan = make(chan globalConnEvent)
	// the channel used to pass channel to build directional communication
	ChanChan = make(chan QueryChan)
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
				eventChan <- INCOMING
			}
		case DECREASE:
			GetLogger().Debug("handleCnt(): connection cnt decreased")
			cnt--
			if cnt < 0 {
				debug.Stack()
				panic("cnt processor was written wrong!")
			} else if cnt == 0 {
				eventChan <- EMPTY
			}
		}
	}
}

// STOPPED->BOOTING->RUNNING<->WAITING->STOPPING->STOPPED
var waitChan chan struct{}

// no we don't need cmdChan, we just make it work immediately
func handleWaitingToRunning() {
	waitChan <- struct{}{}
	_, ok := <-waitChan
	if ok {
		state = RUNNING //happens immediately
	} //else it's too late
}
func handleRunningToWaiting() {
	waitChan = make(chan struct{})
	go waitingThread()
	state = WAITING
}

// TODO: work with daemon
func handleWaitingToStopping() {
	go stoppingThread()
	state = STOPPING
}
func stoppingThread() {
	DaemonChanTX <- struct{}{}
	<-DaemonChanRX
	handleStoppingToStopped()
}
func handleStoppingToStopped() {
	state = STOPPED
}
func bootingThread() {
	DaemonChanTX <- struct{}{}
	<-DaemonChanRX
	handleBootingToRunning()
}

var RunningChan = make(chan struct{})

func handleBootingToRunning() {
	state = RUNNING
	RunningChan <- struct{}{}
}
func handleStoppedToBooting() {
	go bootingThread()
	state = BOOTING
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
	unused := NewStack[int]()
	const (
		// CMD must come first
		CHANCHAN = iota
		EVENTCHAN
		SIZE
	)

	selectCases := make([]reflect.SelectCase, 1)
	//selectCases[CHANCHAN] = reflect.SelectCase{
	//	Dir:  reflect.SelectRecv, // readonly
	//	Chan: reflect.ValueOf(ChanChan),
	//}
	//selectCases[EVENTCHAN] = reflect.SelectCase{
	//	Dir:  reflect.SelectRecv, // readonly
	//	Chan: reflect.ValueOf(eventChan),
	//}
	//selectCases[CMDCHAN] = reflect.SelectCase{
	//	Dir:  reflect.SelectRecv,
	//	Chan: reflect.ValueOf(cmdChan),
	//}
	packagedChan := make(QueryChan)
	selectCases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(packagedChan),
	}
	addChan := func(Chan QueryChan) {
		logger.Debug("addChan(): adding new chan from chanchan")
		nelem := reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(Chan),
		}
		logger.Debug("addChan(): done")
		if unused.IsEmpty() {
			selectCases = append(selectCases, nelem)
		} else {
			selectCases[unused.Pop()] = nelem
		}
	}
	const SIGNAL = 114514
	queryThread := func() {
		prevId := -1
		for {
			id, recv, ok := reflect.Select(selectCases)
			logger.Debugf("queryThread(): recv message from No.%d", id)
			if prevId != -1 {
				logger.Debug("queryThread(): clearing previous")
				selectCases[prevId].Dir = reflect.SelectRecv
				selectCases[prevId].Send = reflect.Value{}
				prevId = -1
			}
			if !ok {
				unused.Push(id)
				continue
			}
			if id != 0 {
				packagedChan <- MCState(recv.Int())
				state, _ := <-packagedChan
				selectCases[id].Dir = reflect.SelectSend
				selectCases[id].Send = reflect.ValueOf(state)
				prevId = id
			}

		}
	}
	go queryThread()
	logger.Debug("handleState(): ready")
	for {
		select {
		case nchan, _ := <-ChanChan:
			addChan(nchan)
			packagedChan <- SIGNAL
			//selectCases[id].Dir = reflect.SelectSend
			//selectCases[id].Send = reflect.ValueOf(make(QueryChan))
			//prevId = id
		case event, _ := <-eventChan:
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
				default:
					panic("handleState():written wrong!" + stateToStr[state])
				}
			case EMPTY:
				switch state {
				case RUNNING:
					handleRunningToWaiting()
				case BOOTING:
				default:
					panic("handleState():written wrong!" + stateToStr[state])
				}
			}
		case <-packagedChan:
			packagedChan <- state
		}
	}

}
