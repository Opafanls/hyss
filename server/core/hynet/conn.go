package hynet

import (
	"context"
	"github.com/Opafanls/hylan/server/constdef"
	"io"
	"net"
	"sync"
	"time"
)

type IHyConn interface {
	Init() error
	SetConfig(netConfig NetConfig, config interface{}) error
	GetConfig(netConfig NetConfig) (data interface{}, exist bool)
	Ctx() context.Context
	Conn() io.ReadWriter
	Flushable
	io.ReadWriteCloser
}

type Flushable interface {
	Flush() error
}

func NewHyConn(conn net.Conn) IHyConn {
	hyConn := &DefaultConn{}
	hyConn.conn = conn
	return hyConn
}

type DefaultConn struct {
	conn   net.Conn
	ctx    context.Context
	config sync.Map
}

func (hyConn *DefaultConn) Init() error {
	return nil
}

func (hyConn *DefaultConn) Conn() io.ReadWriter {
	return hyConn.conn
}

func (hyConn *DefaultConn) Write(data []byte) (int, error) {
	return hyConn.conn.Write(data)
}

func (hyConn *DefaultConn) Read(data []byte) (int, error) {
	return hyConn.conn.Read(data)
}

func (hyConn *DefaultConn) SetConfig(netConfig NetConfig, config interface{}) error {
	switch netConfig {
	case WriteTimeout:
		timeout := config.(time.Time)
		return hyConn.conn.SetWriteDeadline(timeout)
	case ReadTimeout:
		timeout := config.(time.Time)
		return hyConn.conn.SetReadDeadline(timeout)
	default:

	}
	return nil
}

func (hyConn *DefaultConn) GetConfig(netConfig NetConfig) (interface{}, bool) {
	return hyConn.config.Load(netConfig)
}

func (hyConn *DefaultConn) Close() error {
	err := hyConn.conn.Close()
	if err != nil {
		return nil
	}
	return constdef.NewHyError("active close", err)
}

func (hyConn *DefaultConn) Flush() error {
	return nil
}

func (hyConn *DefaultConn) Ctx() context.Context {
	return hyConn.ctx
}
