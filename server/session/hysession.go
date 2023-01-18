package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pb"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/proto"
	"github.com/Opafanls/hylan/server/protocol"
)

type HySessionI interface {
	Cycle() error
	SessionType() constdef.SessionType
	Close() error
	SetHandler(h ProtocolHandler)
	GetConn() hynet.IHyConn
	GetConfig(key string) (interface{}, bool)
	SetConfig(key string, val interface{})
	Base() base.StreamBaseI
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
	protocolSession ProtocolHandler
	sessionType     constdef.SessionType
	base            base.StreamBaseI
	config          map[string]interface{}
	hynet.IHyConn
}

type HySessionSource struct {
	cache pb.CacheRing
	*HySession
}
type HySessionSink struct {
	*HySession
}

func NewHySession(ctx context.Context, sessionType constdef.SessionType, conn hynet.IHyConn, base base.StreamBaseI) HySessionI {
	hySession := &HySession{
		sessCtx:     ctx,
		sessionType: sessionType,
		base:        base,
		config:      make(map[string]interface{}, 2),
	}
	hySession.IHyConn = conn
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

func (hy *HySession) Cycle() error {
	var err error
	defer func() {
		if err != nil {
			log.Errorf(hy.sessCtx, "session closed with err: %+v", err)
		}
		_ = hy.Close()
	}()
	if hy.protocolSession == nil {
		log.Fatalf(hy.sessCtx, "conn handler is nil")
		return fmt.Errorf("conn handler is nil")
	}
	err = hy.protocolSession.OnStart(hy.sessCtx, hy)
	if err != nil {
		return err
	}
	return nil
}

func (hy *HySession) Close() error {
	err := hy.protocolSession.OnStop()
	if err != nil {
		log.Errorf(hy.sessCtx, "session close err: %+v", err)
		return err
	}
	return nil
}

func (hy *HySession) GetConn() hynet.IHyConn {
	return hy.IHyConn
}

func (hy *HySession) SetHandler(h ProtocolHandler) {
	hy.protocolSession = h
}

func (hy *HySession) Base() base.StreamBaseI {
	return hy.base
}

func (hy *HySession) GetConfig(key string) (interface{}, bool) {
	val, exist := hy.config[key]
	return val, exist
}

func (hy *HySession) SetConfig(key string, val interface{}) {
	if key == constdef.ConfigKeySessionBase {
		hy.base = val.(base.StreamBaseI)
		return
	}
	hy.config[key] = val
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

type BaseHandler struct {
}

type ProtocolHandler interface {
	OnInit(ctx context.Context) error                                                      //最基本的资源检查
	OnStart(ctx context.Context, sess HySessionI) error                                    //开始协议通信
	OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess HySessionI)           //开始推流
	OnMedia(ctx context.Context, mediaType protocol.MediaDataType, data interface{}) error //获取到流媒体数据
	OnStop() error                                                                         //关闭通信
}

func (b *BaseHandler) OnInit(ctx context.Context) error {
	return nil
}

func (b *BaseHandler) OnStart(ctx context.Context, session HySessionI) error {
	return nil
}

func (b *BaseHandler) OnMedia(ctx context.Context, mediaType protocol.MediaDataType, data interface{}) error {
	return nil
}

func (b *BaseHandler) OnStop() error {
	return nil
}

func (b *BaseHandler) OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess HySessionI) {
	sess.SetConfig(constdef.ConfigKeySessionBase, info)
	err := event.PushEvent0(event.CreateSession, NewStreamEvent(info, sess))
	if err != nil {
		log.Errorf(ctx, "onPublish %+v failed: %+v", info, err)
	} else {
		log.Infof(ctx, "onPublish success %+v", info)
	}
	return
}

func (hy *HySession) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["stream_type"] = hy.sessionType
	return json.Marshal(m)
}

func NewStreamEvent(info base.StreamBaseI, sess HySessionI) map[string]interface{} {
	m := make(map[string]interface{})
	m[constdef.ConfigKeyStreamBase] = info
	m[constdef.ConfigKeyStreamSess] = sess
	return m
}
