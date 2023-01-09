package session

import (
	"context"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/pb"
	"github.com/Opafanls/hylan/server/proto"
	"github.com/Opafanls/hylan/server/protocol"
)

type HySessionI interface {
	Cycle()
	SessionType() constdef.SessionType
}

type SourceSessionI interface {
	HySessionI
	Push(ctx context.Context, pkt proto.PacketI)
	Pull(ctx context.Context) (proto.PacketI, bool)
	AddSink(arg *proto.SinkArg) HySessionI
}

type SinkSessionI interface {
	HySessionI
}

type HySession struct {
	sessCtx         context.Context
	protocolSession protocol.Handler
	sessionType     constdef.SessionType
}

type HySessionSource struct {
	cache pb.CacheRing
	*HySession
}
type HySessionSink struct {
	*HySession
}

func NewHySession(ctx context.Context, ps protocol.Handler, sessionType constdef.SessionType) HySessionI {
	hySession := &HySession{
		sessCtx:         ctx,
		protocolSession: ps,
		sessionType:     sessionType,
	}
	if sessionType == constdef.SessionTypeSource {
		sourceSession := &HySessionSource{}
		sourceSession.HySession = hySession
		sourceSession.cache = pb.NewRing0(constdef.DefaultCacheSize)
		return sourceSession
	} else if sessionType == constdef.SessionTypeSink {
		sinkSession := &HySessionSink{}
		sinkSession.HySession = hySession
		return sinkSession
	}
	return hySession
}

func (hy *HySession) Cycle() {
	//session := hy.protocolSession
}

func (hy *HySessionSource) AddSink(arg *proto.SinkArg) HySessionI {
	return nil
}

func (hy *HySessionSource) Push(ctx context.Context, pkt proto.PacketI) {
	hy.cache.Push(pkt)
}

func (hy *HySessionSource) Pull(ctx context.Context) (proto.PacketI, bool) {
	data, exist := hy.cache.Pull()
	if !exist {
		return nil, false
	}
	return data.(proto.PacketI), true
}

func (hy *HySession) SessionType() constdef.SessionType {
	return hy.sessionType
}
