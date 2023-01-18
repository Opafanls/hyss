package srv

import (
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/rtmp"
	"github.com/Opafanls/hylan/server/session"
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
	sess := session.NewHySession(conn.Ctx(), constdef.SessionTypeSource, conn, nil)
	var h session.ProtocolHandler
	switch tag {
	case constdef.Rtmp:
		h = rtmp.NewRtmpHandler(sess)
	default:
		log.Errorf(conn.Ctx(), "protocol not found %s", tag)
	}
	sess.SetHandler(h)
	var err error
	defer func() {
		if err != nil {
			log.Errorf(sess.Ctx(), "conn done with err: %+v", err)
		} else {
			log.Infof(sess.Ctx(), "conn done with no err")
		}
		_ = sess.Close()
	}()
	err = sess.Cycle()
}
