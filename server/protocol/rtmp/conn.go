package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/sirupsen/logrus"
	"github.com/yutopp/go-rtmp"
	"io"
	"net"
)

// Server is a RTMP connection.
type Server struct {
	rtmpServer *rtmp.Server
	config     *hynet.TcpListenConfig
}

func NewServer(config *hynet.TcpListenConfig) *Server {
	s := &Server{}
	srv := rtmp.NewServer(&rtmp.ServerConfig{
		OnConnect: func(conn net.Conn) (io.ReadWriteCloser, *rtmp.ConnConfig) {
			ctx := context.Background()
			rt := NewRtmpHandler(ctx, hynet.NewHyConn(conn))
			return conn, &rtmp.ConnConfig{
				Handler: rt.rh,
				ControlState: rtmp.StreamControlStateConfig{
					DefaultBandwidthWindowSize: 6 * 1024 * 1024 / 8,
				},
				Logger: logrus.New(),
			}
		},
	})
	s.rtmpServer = srv
	s.config = config
	return s
}

func (s *Server) Listen() error {
	addr := &net.TCPAddr{
		IP:   net.ParseIP(s.config.Addr),
		Port: s.config.Port,
	}
	ar := addr.String()
	listener, err := net.Listen("tcp", ar)
	if err != nil {
		return err
	}
	ctx := context.Background()
	log.Infof(ctx, "listen rtmp server@%s", ar)
	return s.rtmpServer.Serve(listener)
}

func (s *Server) Init() error {

	return nil
}
