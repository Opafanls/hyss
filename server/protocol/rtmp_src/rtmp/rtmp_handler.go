package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/av"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/session"
	"io"
)

type RtmpHandler struct {
	s             *Server
	svConn        *core.ConnServer
	handleSession session.HySessionI
	isStart       bool
	r             *VirReader
	w             *VirWriter
}

var server = NewRtmpServer()

func NewHandler(session session.HySessionI) session.ProtocolHandler {
	return &RtmpHandler{
		s:             server,
		handleSession: session,
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

func (h *RtmpHandler) OnPublish(sourceArg *session.SourceArg) error {
	h.r = NewVirReader(h.svConn)
	return h.HandleReader()
}

func (h *RtmpHandler) OnSink(sinkArg *session.SinkArg) error {
	h.w = NewVirWriter(h.svConn)
	return nil
}

// Read 读取packet
func (h *RtmpHandler) Read(packet *av.Packet) error {
	return h.handleSession.Write(packet) //source压入packet
}

func (h *RtmpHandler) Write(packet *av.Packet) error { //写成packet
	return h.w.Write(packet)
}

func (h *RtmpHandler) OnStop() error {
	return nil
}

func (h *RtmpHandler) HandleReader() error {
	h.isStart = true
	r := h.r
	for {
		var p = &av.Packet{}
		if !h.isStart {
			h.closeInter()
			return io.EOF
		}
		err := r.Read(p)
		if err != nil {
			h.closeInter()
			h.isStart = false
			return err
		}

		if err := h.handleSession.Write(p); err != nil {
			return err
		}
	}
}

func (h *RtmpHandler) HandleWrite() {

}

func (h *RtmpHandler) closeInter() {

}
