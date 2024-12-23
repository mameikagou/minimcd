package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var proc *exec.Cmd
var DaemonChanRX = make(chan struct{})
var DaemonChanTX = make(chan struct{})

func Stopped() {
	<-DaemonChanTX
	DaemonChanRX <- struct{}{}
	go Booting()
}
func Booting() {
	GetLogger().Info("Starting MC Server")
	arg := strings.Fields(config.StartCommand)
	proc = exec.Command(arg[0], arg[1:]...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	proc.Start()
	DaemonChanRX <- struct{}{}
	GetLogger().Info("enter RUNNING state")
	go Running()

}
func Running() {
	<-DaemonChanTX
	go Stopping()
	DaemonChanRX <- struct{}{}
}
func Stopping() {
	proc.Process.Signal(syscall.SIGINT)
	proc.Wait()
	go Stopped()
	DaemonChanRX <- struct{}{}
}
