package rtmp

import (
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/sirupsen/logrus"
	"github.com/yutopp/go-rtmp"
	"io"
	"net"
	"sync"
)

// Server is a RTMP connection.
type Server struct {
}

type ClientConn struct {
	conn    hynet.IHyConn
	lastErr error
	m       sync.RWMutex
}

func NewServer() {
	srv := rtmp.NewServer(&rtmp.ServerConfig{
		OnConnect: func(conn net.Conn) (io.ReadWriteCloser, *rtmp.ConnConfig) {
			return conn, &rtmp.ConnConfig{
				Handler: h,
				ControlState: rtmp.StreamControlStateConfig{
					DefaultBandwidthWindowSize: 6 * 1024 * 1024 / 8,
				},
				Logger: logrus.New(),
			}
		},
	})
	srv.Serve()
}
