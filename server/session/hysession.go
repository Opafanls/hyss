package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol/container"
	"sync"
	"time"
)

type HySessionI interface {
	Ctx() context.Context
	Cycle() error
	SessionType() constdef.SessionType
	Close() error
	SetHandler(h ProtocolHandler)
	GetConn() hynet.IHyConn
	GetConfig(key constdef.SessionConfigKey) (interface{}, bool)
	SetConfig(key constdef.SessionConfigKey, val interface{})
	Base() base.StreamBaseI
	// Push 把packet压入cache
	Push(ctx context.Context, pkt *model.Packet) error
	// Pull 从cache中取出packet
	Pull(ctx context.Context) (*model.Packet, bool, error)
	// Sink 增加一路sink
	Sink(arg *SinkArg) error
	Stat() *HySessionStat
}

type HySessionStatI interface {
	IncrVideoPkt(num int)
	IncrAudioPkt(num int)
	Running() bool
}

type HySession struct {
	sessCtx     context.Context
	handler     ProtocolHandler
	sessionType constdef.SessionType
	base        base.StreamBaseI
	config      map[constdef.SessionConfigKey]interface{}
	hynet.IHyConn
	cache         *HyDataCache
	hySessionStat *HySessionStat
	l             sync.Mutex
}

func NewHySession(ctx context.Context, sessionType constdef.SessionType, conn hynet.IHyConn, base base.StreamBaseI, kvs ...*model.KV) HySessionI {
	hySession := &HySession{
		sessCtx:       ctx,
		sessionType:   sessionType,
		base:          base,
		config:        make(map[constdef.SessionConfigKey]interface{}, 2+len(kvs)),
		hySessionStat: &HySessionStat{running: true},
	}
	hySession.IHyConn = conn
	hySession.cache = NewHyDataCache([]uint64{
		constdef.DefaultCacheSize,
		16,
	})
	for _, kv := range kvs {
		hySession.config[kv.K] = kv.V
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
	if hy.handler == nil {
		log.Fatalf(hy.sessCtx, "Conn handler is nil")
		return fmt.Errorf("conn handler is nil")
	}
	var info base.StreamBaseI
	info, err = hy.handler.OnStart(hy.sessCtx, hy)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("info is nil")
	}
	hy.base = info
	return hy.handler.OnStreaming(hy.sessCtx, info, hy)
}

func (hy *HySession) Close() error {
	hy.l.Lock()
	defer hy.l.Unlock()
	if !hy.hySessionStat.running {
		return fmt.Errorf("session not running")
	}
	hy.hySessionStat.running = false
	err := hy.handler.OnStop()
	if err != nil {
		log.Errorf(hy.Ctx(), "session close err: %+v", err)
		return err
	}
	err = hy.GetConn().Close()
	if err != nil {
		log.Errorf(hy.Ctx(), "close Conn failed: %+v", err)
	}
	if hy.base != nil {
		st := BaseSessionType(hy)
		if st.IsSink() {
			return event.PushEvent0(hy.sessCtx, event.RemoveSinkSession, hy.base.ID())
		} else {
			return event.PushEvent0(hy.sessCtx, event.RemoveSourceSession, hy.base.ID())
		}
	} else {
		log.Warnf(hy.sessCtx, "hy base is nil")
	}
	return nil
}

func (hy *HySession) GetConn() hynet.IHyConn {
	return hy.IHyConn
}

func (hy *HySession) SetHandler(h ProtocolHandler) {
	hy.handler = h
}

func (hy *HySession) Base() base.StreamBaseI {
	return hy.base
}

func (hy *HySession) GetConfig(key constdef.SessionConfigKey) (interface{}, bool) {
	val, exist := hy.config[key]
	return val, exist
}

func (hy *HySession) SetConfig(key constdef.SessionConfigKey, val interface{}) {
	switch key {
	case constdef.ConfigKeySessionBase:
		hy.base = val.(base.StreamBaseI)
	case constdef.ConfigKeyVideoCodec:
		hy.hySessionStat.videoCodec = val.(constdef.VCodec)
	case constdef.ConfigKeyAudioCodec:
		hy.hySessionStat.audioCodec = val.(constdef.ACodec)
	case constdef.ConfigKeySessionType:
		hy.sessionType = constdef.SessionType(val.(int))
	}
	hy.config[key] = val
}

func (hy *HySession) Sink(arg *SinkArg) error {
	var err error
	sourceStat := arg.Local.Stat()
	sourceSess := arg.Local
	var rw0 container.ReadWriter
	rw, ok := arg.Remote.GetConfig(constdef.ConfigKeySinkRW)
	if !ok {
		return fmt.Errorf("sink ConfigKeySinkRW not set")
	} else {
		if rw0, ok = rw.(container.ReadWriter); !ok {
			return fmt.Errorf("container rw cast failed")
		}
	}
	for sourceStat.running {
		pkt, exist, err := sourceSess.Pull(hy.sessCtx)
		if err != nil {
			return fmt.Errorf("pull packet err: %+v", err)
		}
		if !exist {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		switch pkt.MediaType {
		case constdef.MediaDataTypeVideo:
			if err := rw0.Write(pkt); err != nil {
				if shouldContainer := rw0.OnError(err); shouldContainer {
					log.Warnf(arg.Ctx, "write pkt warn: %+v", err)
					continue
				} else {
					log.Errorf(arg.Ctx, "write pkt failed: %+v", err)
					return err
				}
			}
		case constdef.MediaDataTypeAudio:
		}
	}
	return err
}

func (hy *HySession) Push(ctx context.Context, pkt *model.Packet) error {
	if hy.sessionType != constdef.SessionTypeRtmpSource {
		return constdef.SessionCannotPushMedia
	}
	hy.cache.Push(pkt)
	return nil
}

func (hy *HySession) Pull(ctx context.Context) (*model.Packet, bool, error) {
	if hy.sessionType != constdef.SessionTypeRtmpSource {
		return nil, false, constdef.SessionCannotPullMedia
	}
	data := hy.cache.Pull()
	return data, true, nil
}

func (hy *HySession) SessionType() constdef.SessionType {
	return hy.sessionType
}

func (hy *HySession) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["stream_type"] = hy.sessionType
	if hy.sessionType == constdef.SessionTypeRtmpSource {
		m["video_pkt_recv"] = hy.hySessionStat.videoPktNum
		m["audio_pkt_recv"] = hy.hySessionStat.audioPktNum
		m["videoCodec"] = hy.hySessionStat.videoCodec
		m["audioCodec"] = hy.hySessionStat.audioCodec
	}
	return json.Marshal(m)
}

func (hy *HySession) Stat() *HySessionStat {
	return hy.hySessionStat
}

type BaseHandler struct {
	Ctx  context.Context
	Conn hynet.IHyConn
	Sess HySessionI
	Base base.StreamBaseI
}

type ProtocolHandler interface {
	OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error)        //开始协议通信，握手之类的初始化工作
	OnStreaming(ctx context.Context, info base.StreamBaseI, sess HySessionI) error //开始推流
	//OnMedia(ctx context.Context, w *model.Packet) error                            //获取到流媒体数据
	OnStop() error //关闭通信
}

func (b *BaseHandler) OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error) {
	return nil, nil
}

func (b *BaseHandler) OnMedia(ctx context.Context, w *model.Packet) error {
	source := b.Sess
	mediaData := make([]byte, 0, len(w.Data))
	copy(mediaData, w.Data)
	err := source.Push(ctx, w)
	if err != nil {
		return err
	}
	return nil
}

func (b *BaseHandler) OnStop() error {
	return nil
}

func (b *BaseHandler) OnStreaming(ctx context.Context, info base.StreamBaseI, sess HySessionI) error {
	if sess.SessionType() != constdef.SessionTypeRtmpSource {
		return nil
	}
	sess.SetConfig(constdef.ConfigKeySessionBase, info)
	sessType := BaseSessionType(sess)
	var err error
	if sessType.IsSource() {
		err = event.PushEvent0(ctx, event.CreateSourceSession, NewStreamEvent(info, sess))
	} else {
		sourceID := info.ID()
		sinkID := SinkID(sourceID)
		info.SetParam(base.ParamKeyID, sinkID)
		err = event.PushEvent0(ctx, event.CreateSinkSession, &model.KVStr{K: sourceID, V: sinkID})
	}
	if err != nil {
		log.Errorf(ctx, "onPublish %+v failed: %+v", info, err)
	} else {
		log.Infof(ctx, "onPublish success %+v", info)
	}
	return err
}

func NewStreamEvent(info base.StreamBaseI, sess HySessionI) map[constdef.SessionConfigKey]interface{} {
	m := make(map[constdef.SessionConfigKey]interface{})
	m[constdef.ConfigKeyStreamBase] = info
	m[constdef.ConfigKeyStreamSess] = sess
	return m
}

type HySessionStat struct {
	videoPktNum int
	audioPktNum int
	videoCodec  constdef.VCodec
	audioCodec  constdef.ACodec

	running bool
}

func (stat *HySessionStat) IncrVideoPkt(num int) {
	stat.videoPktNum++
}
func (stat *HySessionStat) IncrAudioPkt(num int) {
	stat.audioPktNum++
}

func (stat *HySessionStat) Running() bool {
	return stat.running
}
