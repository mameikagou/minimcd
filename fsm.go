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
func handleWaitingToStopping() {}
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

	selectCases := make([]reflect.SelectCase, SIZE)
	selectCases[CHANCHAN] = reflect.SelectCase{
		Dir:  reflect.SelectRecv, // readonly
		Chan: reflect.ValueOf(ChanChan),
	}
	selectCases[EVENTCHAN] = reflect.SelectCase{
		Dir:  reflect.SelectRecv, // readonly
		Chan: reflect.ValueOf(eventChan),
	}
	//selectCases[CMDCHAN] = reflect.SelectCase{
	//	Dir:  reflect.SelectRecv,
	//	Chan: reflect.ValueOf(cmdChan),
	//}
	addChan := func(Chan QueryChan) {
		logger.Debug("handleState():adding new chanchan")
		nelem := reflect.SelectCase{
			Dir:  reflect.SelectDefault,
			Chan: reflect.ValueOf(Chan),
		}
		if unused.IsEmpty() {
			selectCases = append(selectCases, nelem)
		} else {
			selectCases[unused.Pop()] = nelem
		}
	}
	logger.Debug("handleState(): ready")
	for {
		id, recv, ok := reflect.Select(selectCases)
		if !ok {
			unused.Push(id)
			continue
		}
		switch id {
		case CHANCHAN:
			nchan := *(*QueryChan)(recv.UnsafePointer())
			addChan(nchan)
		case EVENTCHAN:
			// wait until transformation finishes
			event := globalConnEvent(recv.Int())
			switch event {
			case INCOMING:
				switch state {
				case STOPPED:
				// case STOPPING: // it shouldn't be there, too, should be handled with conn.go
				case WAITING:
					handleWaitingToRunning()
				default:
					panic("handleState():written wrong!")
				}
			case EMPTY:
				switch state {
				case RUNNING:
					handleRunningToWaiting()
				default:
					panic("handleState():written wrong!")
				}
			}
		default:
		}
	}

}
