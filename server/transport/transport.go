package transport

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"io"
)

type Transport interface {
	Close(ctx context.Context, reason string) error
	Recv(ctx context.Context) ([]byte, error)
	Send(ctx context.Context, sendData []byte) error
	GetConn() io.ReadWriter
}

type TcpTransport struct {
	Conn hynet.IHyConn
}

type UdpTransport struct {
}

func (t *TcpTransport) Close(ctx context.Context, reason string) error {
	err := t.Conn.Close()
	log.Infof(ctx, "tcp transport closed %s with err: %+v", reason, err)
	return err
}

func (t *TcpTransport) Recv(ctx context.Context) ([]byte, error) {
	var data = make([]byte, 1024)
	readLen, err := t.Conn.Read(data)
	if err != nil {
		return nil, err
	}
	return data[:readLen], nil
}

func (t *TcpTransport) Send(ctx context.Context, sendData []byte) error {
	sendLen, err := t.Conn.Write(sendData)
	if err != nil {
		return err
	}
	if len(sendData) != sendLen {
		log.Errorf(ctx, "send len not full")
	}
	return nil
}

func (t *TcpTransport) GetConn() io.ReadWriter {
	return t.Conn
}
