package main

import (
	"io"
	"net"
	"sync"
)

func handle(client net.Conn) {
	defer client.Close()
	queryChan := make(QueryChan)
	ChanChan <- queryChan
	proceed := func() {
		server, err := net.Dial("tcp", "127.0.0.1:25565")
		defer server.Close()
		if err != nil {
			GetLogger().Errorf("Failed to connect to MC server: %v", err)
			return
		}
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
		GetLogger().Infof("Connection from %s closed", client.RemoteAddr())
	}
	switch state {
	case RUNNING:
		CntChan <- INCREASE
		defer func() { CntChan <- DECREASE }()
		proceed()
	case STOPPED, WAITING:
		//	client.Write([]byte("Server not ready!"))
		GetLogger().Warnf("Connection queued, server currently at %s state", stateToStr[state])
		CntChan <- INCREASE
		defer func() { CntChan <- DECREASE }()
		<-RunningChan
		proceed()
	default:
		client.Write([]byte("Server not ready!"))
		GetLogger().Warnf("Connection refused, server currently at %s state", stateToStr[state])
	}
}
func Listen() {
	listener, _ := net.Listen("tcp", "0.0.0.0:"+config.Port)
	defer listener.Close()
	GetLogger().Infof("Listening on %s", config.Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			GetLogger().Errorf("ection error: %v", err)
			continue
		}

		GetLogger().Infof("New connection from %s", conn.RemoteAddr())
		go handle(conn)
	}
}
