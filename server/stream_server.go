package server

import (
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/protocol/rtmp"
	"github.com/Opafanls/hylan/server/stream"
	"github.com/Opafanls/hylan/server/task"
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
	task.InitTaskSystem()
}

func (hy *HylanServer) initServer() {
	hy.listeners()
}

func (hy *HylanServer) listeners() {
	listeners := []hynet.ListenServer{
		rtmp.NewServer(&hynet.TcpListenConfig{
			Addr: "",
			Port: 1935,
		}),
	}

	for _, listener := range listeners {
		err := listener.Init()
		if err != nil {
			panic(err)
		}
		err = listener.Start()
		if err != nil {
			panic(err)
		}
	}
}

func (hy *HylanServer) wait() {
	<-hy.stopChan
}
