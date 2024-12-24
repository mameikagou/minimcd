package main

import (
	"io"
	"net"
	"sync"
	"time"
)

type timeoutConn struct {
	conn net.Conn
}

func (c timeoutConn) Read(buf []byte) (int, error) {
	c.conn.SetDeadline(time.Now().Add(time.Duration(config.ConnectTimeout) * time.Second))
	return c.conn.Read(buf)
}
func (c timeoutConn) Write(buf []byte) (int, error) {
	c.conn.SetDeadline(time.Now().Add(time.Duration(config.ConnectTimeout) * time.Second))
	return c.conn.Write(buf)
}

type signalChan chan MCState

var signalChanChan = make(chan signalChan)

// 单条管道->多条管道派发器 goroutine only
var clientSignalChan = NewDynamicMultiChan[MCState](false, 2)

func bridge() {
	for {
		msg, _ := <-CriticalSignalChan
		clientSignalChan.TX <- msg
	}
}
func handle(clientOriginal net.Conn) {
	client := timeoutConn{clientOriginal}
	defer clientOriginal.Close()
	queryChan := make(QueryChan)
	defer close(queryChan)
	QueryChanChan <- queryChan
	proceed := func() {
		server, err := net.Dial("tcp", "127.0.0.1:25565")
		if err != nil {
			GetLogger().Errorf("Failed to connect to MC server: %v", err)
			return
		}
		defer server.Close()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(server, client); err != nil {
				GetLogger().Errorf("Error forwarding client to server: %v", err)
			}
		}()

		// 从服务器到客户端
		go func() {
			defer wg.Done()
			if _, err := io.Copy(client, server); err != nil {
				GetLogger().Errorf("Error forwarding server to client: %v", err)
			}
		}()

		wg.Wait()
		GetLogger().Infof("Connection from %s closed", clientOriginal.RemoteAddr())
	}
	queryChan <- STOPPED
	st, _ := <-queryChan
	switch st {
	case RUNNING:
		CntChan <- INCREASE
		defer func() { CntChan <- DECREASE }()
		proceed()
	case STOPPED, WAITING:
		//	client.Write([]byte("Server not ready!"))
		GetLogger().Infof("Connection queued, server currently at %s state", stateToStr[state])
		CntChan <- INCREASE
		defer func() { CntChan <- DECREASE }()
		curChan := make(chan MCState)
		defer close(curChan)
		clientSignalChan.Add(curChan)
		state, _ := <-curChan
		if state == RUNNING {
			proceed()
		} else {
			client.Write([]byte("You are too late, server dying!"))
			GetLogger().Warnf("Connection refused, server currently at %s state", stateToStr[state])
		}
	default:
		client.Write([]byte("Server not ready!"))
		GetLogger().Warnf("Connection refused, server currently at %s state", stateToStr[state])
	}
}
func Listen() {
	go bridge()
	listener, _ := net.Listen("tcp", "0.0.0.0:"+config.Port)
	defer listener.Close()
	GetLogger().Infof("Listening on %s", config.Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			GetLogger().Errorf("ection error: %v", err)
			continue
		}
		//conn.SetReadDeadline(time.Now().Add(time.Duration(config.ConnectTimeout) * time.Second))
		GetLogger().Infof("New connection from %s", conn.RemoteAddr())
		go handle(conn)
	}
}
