package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"sync"
)

type HySessionI interface {
	Ctx() context.Context
	Cycle() error
	SessionType() base.SessionType
	Close() error
	SetHandler(h ProtocolHandler)
	Handler() ProtocolHandler
	GetConn() hynet.IHyConn
	GetConfig(key base.SessionKey) (interface{}, bool)
	SetConfig(key base.SessionKey, val interface{})
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
	sessCtx context.Context
	handler ProtocolHandler
	base    base.StreamBaseI
	hynet.IHyConn
	cache         *HyDataCache
	hySessionStat *HySessionStat
	l             sync.Mutex
}

func NewHySession(ctx context.Context, conn hynet.IHyConn, baseStreamInfo base.StreamBaseI, kvs ...*model.KV) HySessionI {
	hySession := &HySession{
		sessCtx:       ctx,
		base:          baseStreamInfo,
		hySessionStat: &HySessionStat{running: true},
	}
	hySession.IHyConn = conn
	hySession.cache = NewHyDataCache([]uint64{
		base.DefaultCacheSize,
		16,
	})
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
	log.Infof(hy.sessCtx, "OnStart with info: %+v", info)
	if info == nil {
		return fmt.Errorf("info is nil")
	}
	err = hy.checkValid(info)
	if err != nil {
		return err
	}
	hy.base = info
	sessionTyp := hy.SessionType()
	if sessionTyp.IsSource() {
		return hy.publish()
	} else {
		return hy.play()
	}
}

func (hy *HySession) publish() error {
	ctx := hy.Ctx()
	info := hy.Base()
	var err = event.PushEvent0(ctx, event.OnSessionCreate, NewStreamEvent(info, hy))
	if err != nil {
		return err
	}
	return hy.handler.OnPublish(hy.sessCtx, &SourceArg{})
}

func (hy *HySession) play() error {
	//get source session
	var (
		info  = hy.Base()
		vhost = info.Vhost()
		name  = info.Name()
	)
	stream := DefaultHyStreamManager.GetStream(vhost, name)
	if stream == nil {
		return fmt.Errorf("stream not found")
	}
	return stream.Sink(&SinkArg{
		Ctx:         hy.Ctx(),
		Sink:        &SinkRtmp{},
		SinkSession: hy,
	})
}

func (hy *HySession) checkValid(info base.StreamBaseI) error {
	var (
		vhost   = info.Vhost()
		appName = info.App()
		name    = info.Name()
		id      = info.ID()
	)
	if id == 0 || appName == "" || name == "" || vhost == "" {
		return fmt.Errorf("invalid info")
	}
	return nil
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
	return event.PushEvent0(hy.sessCtx, event.OnSessionDelete, NewStreamEvent(hy.Base(), hy))
}

func (hy *HySession) GetConn() hynet.IHyConn {
	return hy.IHyConn
}

func (hy *HySession) SetHandler(h ProtocolHandler) {
	hy.handler = h
}

func (hy *HySession) Handler() ProtocolHandler {
	return hy.handler
}

func (hy *HySession) Base() base.StreamBaseI {
	return hy.base
}

func (hy *HySession) GetConfig(key base.SessionKey) (interface{}, bool) {
	return hy.base.GetParam(key)
}

func (hy *HySession) SetConfig(key base.SessionKey, val interface{}) {
	switch key {
	case base.ConfigKeySessionBase:
		hy.base = val.(base.StreamBaseI)
	case base.ConfigKeyVideoCodec:
		hy.hySessionStat.videoCodec = val.(base.VCodec)
	case base.ConfigKeyAudioCodec:
		hy.hySessionStat.audioCodec = val.(base.ACodec)
		hy.base.SetParam(key, val)
	}
}

func (hy *HySession) Sink(arg *SinkArg) error {
	return nil
}

func (hy *HySession) Push(ctx context.Context, pkt *model.Packet) error {
	return nil
}

func (hy *HySession) Pull(ctx context.Context) (*model.Packet, bool, error) {
	return nil, true, nil
}

func (hy *HySession) SessionType() base.SessionType {
	v, e := hy.base.GetParam(base.SessionInitParamKeyStreamType)
	if !e {
		return base.SessionTypeInvalid
	}
	return v.(base.SessionType)
}

func (hy *HySession) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
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
	OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error) //开始协议通信，握手之类的初始化工作
	OnPublish(ctx context.Context, sourceArg *SourceArg) error              //开始推流
	OnSink(ctx context.Context, sinkArg *SinkArg) error                     //开始拉流
	OnStop() error                                                          //关闭通信
}

type PlayHandler interface {
	OnPlay()
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

func (b *BaseHandler) OnSink(ctx context.Context, arg *SinkArg) error {
	return nil
}

func (b *BaseHandler) OnPublish(ctx context.Context, arg *SourceArg) error {
	return nil
}

func NewStreamEvent(info base.StreamBaseI, sess HySessionI) map[base.SessionKey]interface{} {
	m := make(map[base.SessionKey]interface{})
	m[base.ConfigKeyStreamBase] = info
	m[base.ConfigKeyStreamSess] = sess
	return m
}

type HySessionStat struct {
	videoPktNum int
	audioPktNum int
	videoCodec  base.VCodec
	audioCodec  base.ACodec

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
