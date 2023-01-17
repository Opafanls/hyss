package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/task"
)

// Server is a RTMP connection.
type Server struct {
	ctx     context.Context
	config  *hynet.TcpListenConfig
	running bool
	*hynet.TcpServer
}

func NewServer(config *hynet.TcpListenConfig) *Server {
	s := &Server{}
	s.config = config
	return s
}

func (s *Server) Start() error {
	s.TcpServer.ConnHandler = s
	return s.TcpServer.Start()
}

func (s *Server) Init() error {
	s.ctx = log.GetCtxWithLogID(context.Background(), "RTMP_SERVER")
	tcpServer := hynet.NewTcpServer(s.ctx, s.config.Addr, s.config.Port)
	s.TcpServer = tcpServer
	if err := s.TcpServer.Init(); err != nil {
		return err
	}
	s.running = true
	return nil
}

func (s *Server) HandleConn(conn hynet.IHyConn) {
	//accept tcp connection
	task.SubmitTask0(s.ctx, func() {
		rtmpHandler := NewRtmpHandler(log.GetCtxWithLogID(s.ctx, ""), conn)
		var err error
		defer func() {
			if err != nil {
				log.Errorf(s.ctx, "conn done with err: %+v", err)
			} else {
				log.Infof(s.ctx, "conn done with no err")
			}
			_ = rtmpHandler.OnClose()
		}()
		err = rtmpHandler.OnInit(s.ctx)
	})
}
