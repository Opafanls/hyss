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

var DefaultHyStreamManager HyStreamManagerI

type HyStreamManagerI interface {
	StreamFilter(interface{}) (interface{}, error)
	AddStream(hyStream HyStreamI)
	RemoveStream(streamBaseID string)
	GetStreamByID(id string) HyStreamI
}

type HyStreamManager struct {
	rwLock    *sync.RWMutex
	streamMap map[string]HyStreamI
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
	HyStreamManager.streamMap = make(map[string]HyStreamI)
	return HyStreamManager
}

func (streamManager *HyStreamManager) AddStream(hyStream HyStreamI) {
	id := hyStream.Base().ID()
	streamManager.rwLock.Lock()
	streamManager.streamMap[id] = hyStream
	streamManager.rwLock.Unlock()
}

func (streamManager *HyStreamManager) RemoveStream(streamBaseID string) {
	streamManager.rwLock.Lock()
	delete(streamManager.streamMap, streamBaseID)
	streamManager.rwLock.Unlock()
}

// StreamFilter for biz
func (streamManager *HyStreamManager) StreamFilter(filter interface{}) (interface{}, error) {
	if filter == nil {
		//ret all streams
		r := make(map[string]interface{}, len(streamManager.streamMap))
		for k, v := range streamManager.streamMap {
			r[k] = v
		}
		return r, nil
	}
	return nil, nil
}

func (streamManager *HyStreamManager) GetStreamByID(id string) HyStreamI {
	streamManager.rwLock.RLock()
	r := streamManager.streamMap[id]
	streamManager.rwLock.RUnlock()
	return r
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
		m := data.(map[constdef.SessionConfigKey]interface{})
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
