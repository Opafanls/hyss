package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/av"
	"github.com/Opafanls/hylan/server/core/cache"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol"
	"sync"
)

type HySessionI interface {
	Ctx() context.Context
	Cycle() error
	SessionType() base.SessionType
	Close() error
	SetHandler(h ProtocolHandler) //不同的媒体协议
	Handler() ProtocolHandler
	GetConn() hynet.IHyConn
	GetConfig(key base.SessionKey) (interface{}, bool)
	SetConfig(key base.SessionKey, val interface{})
	Base() base.StreamBaseI
	Sink(*SinkArg) error
	av.PacketWriter
	av.PacketReader
	Stat() *HySessionStat
}

type HySessionStatI interface {
	IncrVideoPkt(num int)
	IncrAudioPkt(num int)
	Running() bool
}

type HySession struct {
	sessCtx       context.Context
	handler       ProtocolHandler
	base          base.StreamBaseI
	conn          hynet.IHyConn
	hySessionStat *HySessionStat
	l             sync.Mutex
	pktSaver      *protocol.PacketReceiver
	pktCache      *cache.Cache
	src           HyStreamI
}

func NewHySession(ctx context.Context, conn hynet.IHyConn, baseStreamInfo base.StreamBaseI, kvs ...*model.KV) HySessionI {
	hySession := &HySession{
		sessCtx:       ctx,
		base:          baseStreamInfo,
		hySessionStat: &HySessionStat{IsRunning: true},
	}
	hySession.conn = conn
	hySession.pktSaver = protocol.NewPacketReceiver(hySession.handleMedia)
	hySession.pktCache = cache.NewCache()
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
		return DefaultHyStreamManager.Publish(hy)
	} else {
		return DefaultHyStreamManager.Sink(hy)
	}
}

func (hy *HySession) handleMedia(pkt *av.Packet) {
	if hy.SessionType().IsSource() {
		//is source
		pktCache := hy.pktCache
		pktCache.Write(pkt)
		source := hy.src
		if source != nil {
			source.RangeSinks(func(id int64, session HySessionI) {
				stat := session.Stat()
				if !stat.Init {
					if err := pktCache.Send(session); err != nil {
						source.RmSink(id)
					}
					stat.Init = true
				} else {
					newPacket := pkt
					if err := session.Write(newPacket); err != nil {
						source.RmSink(id)
					}
				}
			})
		}
	} else {
		//is sink
		err := hy.handler.Write(pkt)
		if err != nil {
			log.Infof(hy.Ctx(), "handler write packet err: %+v", err)
		}
	}
}

func (hy *HySession) Sink(arg *SinkArg) error {
	if err := hy.Handler().OnSink(arg); err != nil {
		return err
	}
	for {
		pkt := &av.Packet{}
		err := hy.Read(pkt)
		if err != nil {
			return err
		}
		if err = hy.Handler().Write(pkt); err != nil {
			return err
		}
	}
}

func (hy *HySession) Write(pkt *av.Packet) error {
	hy.pktSaver.Push(pkt)
	return nil
}

func (hy *HySession) Read(pull *av.Packet) error {
	pulled := hy.pktSaver.Pull()
	if pulled == nil {
		return fmt.Errorf("invalid packet")
	}
	pulled.CopyTo(pull)
	return nil
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
	if !hy.hySessionStat.Running() {
		return fmt.Errorf("session not running")
	}
	hy.hySessionStat.IsRunning = false
	err := hy.handler.OnStop()
	if err != nil {
		log.Errorf(hy.Ctx(), "session close err: %+v", err)
		return err
	}
	err = hy.GetConn().Close()
	if err != nil {
		log.Errorf(hy.Ctx(), "close Conn failed: %+v", err)
	}
	hy.pktSaver.Stop()
	return event.PushEvent0(hy.sessCtx, event.OnSessionDelete, NewStreamEvent(hy.Base(), hy))
}

func (hy *HySession) GetConn() hynet.IHyConn {
	return hy.conn
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
		hy.hySessionStat.VideoCodec = val.(base.VCodec)
	case base.ConfigKeyAudioCodec:
		hy.hySessionStat.AudioCodec = val.(base.ACodec)
	case base.StreamInitSetSourceStream:
		hy.src = val.(HyStreamI)
	default:
		hy.base.SetParam(key, val)
	}
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
	m["stat"] = hy.Stat()
	m["base"] = hy.base
	return json.Marshal(m)
}

func (hy *HySession) Stat() *HySessionStat {
	return hy.hySessionStat
}

type ProtocolHandler interface {
	OnStart(ctx context.Context, sess HySessionI) (base.StreamBaseI, error) //开始协议通信，握手之类的初始化工作
	OnStop() error
	ProtocolSourceHandler
	ProtocolSinkHandler //关闭通信
}

type ProtocolSourceHandler interface {
	OnPublish(sourceArg *SourceArg) error //开始推流
	av.PacketReader
}

type ProtocolSinkHandler interface {
	// OnSink sink
	OnSink(sinkArg *SinkArg) error
	av.PacketWriter
}

func NewStreamEvent(info base.StreamBaseI, sess HySessionI) map[base.SessionKey]interface{} {
	m := make(map[base.SessionKey]interface{})
	m[base.ConfigKeyStreamBase] = info
	m[base.ConfigKeyStreamSess] = sess
	return m
}

type HySessionStat struct {
	Init        bool
	VideoPktNum int
	AudioPktNum int
	VideoCodec  base.VCodec
	AudioCodec  base.ACodec
	IsRunning   bool
}

func (stat *HySessionStat) IncrVideoPkt(num int) {
	stat.VideoPktNum++
}
func (stat *HySessionStat) IncrAudioPkt(num int) {
	stat.AudioPktNum++
}

func (stat *HySessionStat) Running() bool {
	return stat.IsRunning
}
