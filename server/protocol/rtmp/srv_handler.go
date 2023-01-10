package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
)

type Handler struct {
	ctx                context.Context
	conn               hynet.IHyConn
	rtmpMessageHandler *rtmpMessageHandler
}

type rtmpMessageHandler struct {
	handshake *handshake
}

func NewRtmpHandler(ctx context.Context, conn hynet.IHyConn) *Handler {
	rtmpHandler := &rtmpMessageHandler{}
	rtmpHandler.handshake = NewHandshake()
	h := &Handler{ctx: ctx, conn: conn, rtmpMessageHandler: rtmpHandler}
	return h
}

func (h *Handler) OnInit() {
	var err error
	defer func() {
		if err != nil {
			log.Errorf(h.ctx, "conn done with err: %+v", err)
		} else {
			log.Infof(h.ctx, "conn done with no err")
		}
		_ = h.conn.Close()
	}()
	err = h.handshake()
	if err != nil {
		return
	}
}

func (h *Handler) handshake() error {
	return h.rtmpMessageHandler.handshake.handshake(h.conn)
}

func (h *Handler) createStream() {

}

func (h *Handler) messageLoop() {

}
