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
	"github.com/Opafanls/hylan/server/model"
)

type HySessionI interface {
	Ctx() context.Context
	Cycle() error
	SessionType() constdef.SessionType
	Close() error
	SetHandler(h ProtocolHandler)
	GetConn() hynet.IHyConn
	GetConfig(key string) (interface{}, bool)
	SetConfig(key string, val interface{})
	Base() base.StreamBaseI
	//biz
	Push(ctx context.Context, pkt *model.Packet) *constdef.HyError
	Pull(ctx context.Context) (*model.Packet, bool, *constdef.HyError)
	AddSink(arg *model.SinkArg) HySessionI
}

type HySession struct {
	sessCtx         context.Context
	protocolSession ProtocolHandler
	sessionType     constdef.SessionType
	base            base.StreamBaseI
	config          map[string]interface{}
	hynet.IHyConn
	cache pb.CacheRing
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
		hySession.cache = pb.NewRing0(constdef.DefaultCacheSize)
	} else if sessionType == constdef.SessionTypeSink {

	}
	return hySession
}

func (hy *HySession) Ctx() context.Context {
	return hy.sessCtx
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
	var info base.StreamBaseI
	info, err = hy.protocolSession.OnStart(hy.sessCtx, hy)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("info is nil")
	}
	hy.base = info
	return hy.protocolSession.OnStreamPublish(hy.sessCtx, info, hy)
}

func (hy *HySession) Close() error {
	err := hy.protocolSession.OnStop()
	if err != nil {
		log.Errorf(hy.Ctx(), "session close err: %+v", err)
		return err
	}
	err = hy.GetConn().Close()
	if err != nil {
		log.Errorf(hy.Ctx(), "close conn failed: %+v", err)
	}
	if hy.base != nil {
		return event.PushEvent0(event.RemoveSession, hy.base.ID())
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

func (hy *HySession) AddSink(arg *model.SinkArg) HySessionI {
	return nil
}

func (hy *HySession) Push(ctx context.Context, pkt *model.Packet) *constdef.HyError {
	if hy.sessionType != constdef.SessionTypeSource {
		return constdef.SessionCannotPushMedia
	}
	hy.cache.Push(pkt)
	return nil
}

func (hy *HySession) Pull(ctx context.Context) (*model.Packet, bool, *constdef.HyError) {
	if hy.sessionType != constdef.SessionTypeSource {
		return nil, false, constdef.SessionCannotPullMedia
	}
	data, exist := hy.cache.Pull()
	if !exist {
		return nil, false, nil
	}
	return data.(*model.Packet), true, nil
}

func (hy *HySession) SessionType() constdef.SessionType {
	return hy.sessionType
}

type BaseHandler struct {
}

type ProtocolHandler interface {
	OnInit(ctx context.Context) error                                                  //最基本的资源检查
	OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error)            //开始协议通信，握手之类的初始化工作
	OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess HySessionI) error //开始推流
	OnMedia(ctx context.Context, w *model.Packet) error                                //获取到流媒体数据
	OnStop() error                                                                     //关闭通信
}

func (b *BaseHandler) OnInit(ctx context.Context) error {
	return nil
}

func (b *BaseHandler) OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error) {
	return nil, nil
}

func (b *BaseHandler) OnMedia(ctx context.Context, w *model.Packet) error {
	return nil
}

func (b *BaseHandler) OnStop() error {
	return nil
}

//func (b *BaseHandler) OnPrepared(base.StreamBaseI) error {
//	return nil
//}

func (b *BaseHandler) OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess HySessionI) error {
	sess.SetConfig(constdef.ConfigKeySessionBase, info)
	err := event.PushEvent0(event.CreateSession, NewStreamEvent(info, sess))
	if err != nil {
		log.Errorf(ctx, "onPublish %+v failed: %+v", info, err)
	} else {
		log.Infof(ctx, "onPublish success %+v", info)
	}
	return err
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
