package protocol

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
)

type Handler interface {
	OnMedia(ctx context.Context, mediaType MediaDataType, data interface{}) error
	OnClose() error
}

type TcpHandler struct {
	conn hynet.IHyConn
	ctx  context.Context
}

func (tcp *TcpHandler) OnMedia(ctx context.Context, mediaType MediaDataType, data interface{}) error {

	return nil
}

func (tcp *TcpHandler) OnClose() error {
	return tcp.conn.Close()
}

func (tcp *TcpHandler) Ctx() context.Context {
	return tcp.ctx
}

func NewTcpHandler(ctx context.Context, conn hynet.IHyConn) *TcpHandler {
	tcpHandler := &TcpHandler{}
	tcpHandler.conn = conn
	tcpHandler.ctx = ctx
	return tcpHandler
}
