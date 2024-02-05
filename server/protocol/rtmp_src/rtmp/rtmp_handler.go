package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/session"
)

type RtmpHandler struct {
	s      *Server
	svConn *core.ConnServer
}

var server = NewRtmpServer(NewRtmpStream(), nil)

func NewHandler() session.ProtocolHandler {
	return &RtmpHandler{
		s: server,
	}
}

func (h *RtmpHandler) OnStart(ctx context.Context, sess session.HySessionI) (base.StreamBaseI, error) {
	s := h.s
	rawConn := sess.GetConn().Conn()
	conn := core.NewConn(rawConn, 4*1024)
	err := s.connInit(conn)
	if err != nil {
		return nil, err
	}
	svConn := s.connServer
	h.svConn = svConn
	err = s.oncheck()
	if err != nil {
		return nil, err
	}
	var (
		rtmpUrl = svConn.Url
		name    = svConn.Name
		appname = svConn.Appname
	)
	log.Infof(ctx, "onCheck ok: %+v", svConn)
	sInfo, err := base.NewBase(rtmpUrl)
	if err != nil {
		return nil, err
	}
	sInfo.SetParam(base.SessionInitParamKeyName, name)
	sInfo.SetParam(base.SessionInitParamKeyApp, appname)
	sInfo.SetParam(base.SessionInitParamKeyStreamType, h.sessionType())
	return sInfo, nil
}

func (h *RtmpHandler) sessionType() base.SessionType {
	if h.svConn.IsPublisher() {
		return base.SessionTypeRtmpSource
	}
	return base.SessionTypeRtmpSink
}

func (h *RtmpHandler) OnPublish(ctx context.Context, sourceArg *session.SourceArg) error {
	return h.s.Publish(ctx, &session.SourceArg{})
}

func (h *RtmpHandler) OnSink(ctx context.Context, sinkArg *session.SinkArg) error {
	return h.s.OnSink(ctx, sinkArg)
}

func (h *RtmpHandler) OnStop() error {
	return nil
}
