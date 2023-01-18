package srv

import (
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
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
	task.InitTaskSystem() //init first
	event.Init()
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
	listeners := []hynet.ListenServer{
		NewServer(&hynet.TcpListenConfig{
			Ip:   "",
			Port: 1935,
		}),
		NewHttpServer(&hynet.HttpServeConfig{
			Ip:   "",
			Port: 6789,
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
