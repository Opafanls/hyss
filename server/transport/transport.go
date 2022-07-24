package transport

import (
	"context"
	"io"
	"net"
)

type Transport interface {
	Close(ctx context.Context, reason string) error
	Recv(ctx context.Context, recvData []byte) (int64, error)
	Send(ctx context.Context, sendData []byte, offset int64, len int64) (int64, error)
	GetConn() io.ReadWriteCloser
}

type TcpTransport struct {
	Conn net.Conn
}

type UdpTransport struct {
}

func (t *TcpTransport) Close(ctx context.Context, reason string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TcpTransport) Recv(ctx context.Context, recvData []byte) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TcpTransport) Send(ctx context.Context, sendData []byte, offset int64, len int64) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TcpTransport) GetConn() io.ReadWriteCloser {
	return t.Conn
}
