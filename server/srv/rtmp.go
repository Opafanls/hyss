package srv

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
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

func (s *Server) Serve(tag string) error {
	return s.TcpServer.Serve(tag)
}

func (s *Server) Init(ch hynet.ConnHandler) error {
	s.ctx = log.GetCtxWithLogID(context.Background(), "RTMP_SERVER")
	tcpServer := hynet.NewTcpServer(s.ctx, s.config.Ip, s.config.Port, "rtmp")
	s.TcpServer = tcpServer
	s.ConnHandler = ch
	if err := s.TcpServer.Init(ch); err != nil {
		return err
	}
	return nil
}
