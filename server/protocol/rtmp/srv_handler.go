package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol"
	"github.com/yutopp/go-rtmp"
	"github.com/yutopp/go-rtmp/message"
	"io"
)

type Handler struct {
	*protocol.TcpHandler

	rh *rtmpHandler
}

type rtmpHandler struct {
	rtmpConn *rtmp.Conn
	ctx      context.Context
}

func NewRtmpHandler(ctx context.Context, conn hynet.IHyConn) *Handler {
	h := &Handler{}
	h.TcpHandler = protocol.NewTcpHandler(ctx, conn)
	h.rh = &rtmpHandler{}
	return h
}

func (h *rtmpHandler) OnServe(conn *rtmp.Conn) {
	h.rtmpConn = conn
}

func (h *rtmpHandler) OnConnect(timestamp uint32, cmd *message.NetConnectionConnect) error {
	return nil
}

func (h *rtmpHandler) OnCreateStream(timestamp uint32, cmd *message.NetConnectionCreateStream) error {
	return nil
}

func (h *rtmpHandler) OnReleaseStream(timestamp uint32, cmd *message.NetConnectionReleaseStream) error {
	return nil
}

func (h *rtmpHandler) OnDeleteStream(timestamp uint32, cmd *message.NetStreamDeleteStream) error {
	return nil
}

func (h *rtmpHandler) OnPublish(ctx *rtmp.StreamContext, timestamp uint32, cmd *message.NetStreamPublish) error {
	return nil
}

func (h *rtmpHandler) OnPlay(ctx *rtmp.StreamContext, timestamp uint32, cmd *message.NetStreamPlay) error {
	return nil
}

func (h *rtmpHandler) OnFCPublish(timestamp uint32, cmd *message.NetStreamFCPublish) error {
	return nil
}

func (h *rtmpHandler) OnFCUnpublish(timestamp uint32, cmd *message.NetStreamFCUnpublish) error {
	return nil
}

func (h *rtmpHandler) OnSetDataFrame(timestamp uint32, data *message.NetStreamSetDataFrame) error {
	return nil
}

func (h *rtmpHandler) OnAudio(timestamp uint32, payload io.Reader) error {
	return nil
}

func (h *rtmpHandler) OnVideo(timestamp uint32, payload io.Reader) error {
	return nil
}

func (h *rtmpHandler) OnUnknownMessage(timestamp uint32, msg message.Message) error {
	return nil
}

func (h *rtmpHandler) OnUnknownCommandMessage(timestamp uint32, cmd *message.CommandMessage) error {
	return nil
}

func (h *rtmpHandler) OnUnknownDataMessage(timestamp uint32, data *message.DataMessage) error {
	return nil
}

func (h *rtmpHandler) OnClose() {
	err := h.rtmpConn.Close()
	if err != nil {
		log.Errorf(h.ctx, "onclose err: %+v", err)
	}
}
