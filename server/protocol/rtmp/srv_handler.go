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
	handshake   *handshake
	chunkStream *chunkStream
}

func NewRtmpHandler(ctx context.Context, conn hynet.IHyConn) *Handler {
	rtmpHandler := &rtmpMessageHandler{}
	rtmpHandler.handshake = newHandshake()
	rtmpHandler.chunkStream = newChunkStream(conn)
	h := &Handler{ctx: ctx, conn: conn, rtmpMessageHandler: rtmpHandler}
	return h
}

func (h *Handler) OnInit() error {
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
		return err
	}
	return h.messageLoop()
}

func (h *Handler) handshake() error {
	return h.rtmpMessageHandler.handshake.handshake(h.conn)
}

func (h *Handler) messageLoop() error {
	return h.rtmpMessageHandler.chunkStream.decodeChunkStream()
}
