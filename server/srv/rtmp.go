package srv

import (
	"context"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/rtmp"
	"github.com/Opafanls/hylan/server/session"
	"github.com/Opafanls/hylan/server/task"
)

// Server is a RTMP connection.
type Server struct {
	ctx    context.Context
	config *hynet.TcpListenConfig
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
	tcpServer := hynet.NewTcpServer(s.ctx, s.config.Ip, s.config.Port)
	s.TcpServer = tcpServer
	if err := s.TcpServer.Init(); err != nil {
		return err
	}
	return nil
}

func (s *Server) HandleConn(conn hynet.IHyConn) {
	//accept tcp connection
	task.SubmitTask0(s.ctx, func() {
		sess := session.NewHySession(s.ctx, constdef.SessionTypeSource, conn, nil)
		rh := rtmp.NewRtmpHandler(sess)
		sess.SetHandler(rh)
		var err error
		defer func() {
			if err != nil {
				log.Errorf(s.ctx, "conn done with err: %+v", err)
			} else {
				log.Infof(s.ctx, "conn done with no err")
			}
			_ = sess.Close()
		}()
		err = sess.Cycle()
	})
}
