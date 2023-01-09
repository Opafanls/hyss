package server

import (
	"github.com/Opafanls/hylan/server/protocol"
	"github.com/Opafanls/hylan/server/protocol/rtmp"
	"github.com/Opafanls/hylan/server/stream"
)

type HylanServer struct {
	stopChan chan struct{}
}

func NewHylanServer() *HylanServer {
	hylanServer := &HylanServer{}
	hylanServer.stopChan = make(chan struct{})
	return hylanServer
}

func (hy *HylanServer) Start() {
	hy.initBase()
	hy.initServer()
	hy.wait()
}

func (hy *HylanServer) initBase() {
	stream.InitHyStreamManager()
}

func (hy *HylanServer) initServer() {
	hy.listeners()
}

func (hy *HylanServer) listeners() {
	listeners := []protocol.ListenServer{
		rtmp.NewRtmp(&protocol.ListenArg{
			Ip: "0.0.0.0", Port: 1935,
		}),
	}

	for _, listener := range listeners {
		listener.Listen()
	}
}

func (hy *HylanServer) wait() {
	<-hy.stopChan
}
