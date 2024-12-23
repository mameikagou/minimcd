package main

import (
	"reflect"
	"runtime/debug"
)

type MCState int

const (
	STOPPED MCState = iota
	RUNNING
	BOOTING
	WAITING
	STOPPING
)
const QUERY = 114514

type CntEvent int

const (
	INCREASE CntEvent = 1
	DECREASE CntEvent = -1
)

type StateEvent int

const (
	INCOMING StateEvent = iota
	EMPTY
)

type QueryChan chan MCState

var (
	state MCState = STOPPED
	// we don't need mutex for cnt because with channel it's guaranteed to be processed sequently
	CntChan   = make(chan CntEvent)
	eventChan = make(chan StateEvent)
	// the channel used to pass channel to build directional communication
	ChanChan = make(chan QueryChan)
)
var cnt int

func InitState() {
	go handleCnt()
	go handleEvent()
}
func handleCnt() {
	for {
		cntEvent := <-CntChan
		switch cntEvent {
		case INCREASE:
			cnt++
			if cnt == 1 { //which means it is 0->1
				eventChan <- INCOMING
			}
		case DECREASE:
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
func handleEvent() { //handles both write and read
	var chanList []QueryChan
	// TODO: create chanList vector
	for {

	}

}
