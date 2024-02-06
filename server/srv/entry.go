package srv

import (
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp"
	"github.com/Opafanls/hylan/server/session"
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
	session.InitHyStreamManager()
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
			Port: 8080,
		}),
	}

	for _, listener := range listeners {
		err := listener.Init(hy)
		if err != nil {
			panic(err)
		}
		err = listener.Serve(listener.Name())
		if err != nil {
			panic(err)
		}
	}
}

func (hy *HylanServer) wait() {
	<-hy.stopChan
}

// ServeConn 所有tcp连接相关的请求都会统一打到这里
func (hy *HylanServer) ServeConn(tag string, conn hynet.IHyConn) {
	task.SubmitTask0(conn.Ctx(), func() {
		sess := session.NewHySession(conn.Ctx(), conn, base.NewEmptyBase())
		var h session.ProtocolHandler
		switch tag {
		case base.Rtmp:
			h = rtmp.NewHandler(sess)
		default:
			log.Errorf(conn.Ctx(), "protocol not found %s", tag)
			return
		}
		session.Serve(sess, h)
	})
}
