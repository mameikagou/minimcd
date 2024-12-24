package main

type GenericChan chan int
type QueryChan chan MCState
type SignalChan chan MCState

var (
	CntChan       = make(chan CntEvent)
	QueryChanChan = make(chan QueryChan)
)
var CriticalSignalChan = make(chan MCState)

var DaemonChanRX = make(chan struct{})
var DaemonChanTX = make(chan struct{})
