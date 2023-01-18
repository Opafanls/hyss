package stream

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/session"
	"sync"
)

var DefaultHyStreamManager *HyStreamManager

type HyStreamManager struct {
	rwLock    *sync.RWMutex
	streamMap map[string]*HyStream
}

type streamHandler struct {
}

func InitHyStreamManager() {
	DefaultHyStreamManager = NewHyStreamManager()
	err := event.RegisterEventHandler(&streamHandler{})
	if err != nil {
		log.Fatalf(context.Background(), "register event handler err: %+v", err)
	}
}

func NewHyStreamManager() *HyStreamManager {
	HyStreamManager := &HyStreamManager{}
	HyStreamManager.rwLock = &sync.RWMutex{}
	HyStreamManager.streamMap = make(map[string]*HyStream)
	return HyStreamManager
}

func (streamManager *HyStreamManager) AddStream(hyStream *HyStream) {
	id := hyStream.StreamBase.ID()
	streamManager.rwLock.Lock()
	streamManager.streamMap[id] = hyStream
	streamManager.rwLock.Unlock()
}

func (streamManager *HyStreamManager) RemoveStream(streamBaseID string) {
	streamManager.rwLock.Lock()
	delete(streamManager.streamMap, streamBaseID)
	streamManager.rwLock.Unlock()
}

func (streamManager *HyStreamManager) StreamFilter(filter map[string]string) interface{} {
	if filter == nil {
		//ret all streams
		r := make(map[string]*HyStream, len(streamManager.streamMap))
		for k, v := range streamManager.streamMap {
			r[k] = v
		}
		return r
	}
	return nil
}

func (s *streamHandler) Event() []event.HyEvent {
	return []event.HyEvent{
		event.CreateSession,
		event.RemoveSession,
	}
}

func (s *streamHandler) Handle(e event.HyEvent, data interface{}) {
	switch e {
	case event.CreateSession:
		m := data.(map[string]interface{})
		baseI := m[constdef.ConfigKeyStreamBase].(base.StreamBaseI)
		sess := m[constdef.ConfigKeyStreamSess].(session.HySessionI)
		stream := NewHyStream(baseI, sess)
		DefaultHyStreamManager.AddStream(stream)
		break
	case event.RemoveSession:
		DefaultHyStreamManager.RemoveStream(data.(string))
		break
	default:
		log.Errorf(context.Background(), "event not handle %d %v", e, data)
	}
}
