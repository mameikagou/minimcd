package main

type CntEvent int

const (
	INCREASE CntEvent = 1
	DECREASE CntEvent = -1
)

type MCState int

const (
	STOPPED MCState = iota
	RUNNING
	BOOTING
	WAITING
	STOPPING
	SIZE
)

var stateToStr = [SIZE]string{"STOPPED", "RUNNING", "BOOTING", "WAITING", "STOPPING"}
