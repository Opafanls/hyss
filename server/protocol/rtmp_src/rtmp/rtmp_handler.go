package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/session"
)

type Handler struct {
	Session  session.HySessionI
	rtmpConn *core.Conn
}

func (h *Handler) OnInit(ctx context.Context) error {
	conn := core.NewConn(h.Session.GetConn().Conn(), 4*1024)

	h.rtmpConn = conn
	return nil
}

func (h *Handler) OnStart(ctx context.Context, sess session.HySessionI) (base.StreamBaseI, error) {
	return nil, nil
}

func (h *Handler) OnStreaming(ctx context.Context, info base.StreamBaseI, sess session.HySessionI) error {

}

func (h *Handler) OnStop() error {

}
