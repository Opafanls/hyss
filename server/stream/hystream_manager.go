package stream

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/session"
	"github.com/Opafanls/hylan/server/util"
	"sync"
)

var DefaultHyStreamManager HyStreamManagerI

type HyStreamManagerI interface {
	StreamFilter(func(map[string]HyStreamI)) error
	AddStream(hyStream HyStreamI)
	RemoveStream(streamBaseID string)
	RemoveStreamSink(source, sink string) error
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
func (streamManager *HyStreamManager) StreamFilter(filter func(map[string]HyStreamI)) error {
	if filter == nil {
		//ret all streams
		return fmt.Errorf("filter is nil")
	}
	streamM := util.Copy(streamManager.streamMap)
	filter(streamM.(map[string]HyStreamI))
	return nil
}

func (streamManager *HyStreamManager) GetStreamByID(id string) HyStreamI {
	streamManager.rwLock.RLock()
	r := streamManager.streamMap[id]
	streamManager.rwLock.RUnlock()
	return r
}

func (streamManager *HyStreamManager) RemoveStreamSink(source, sink string) error {
	s := streamManager.GetStreamByID(source)
	if s == nil {
		return base.NewHyFunErr("RemoveStreamSink", fmt.Errorf("source not found"))
	}
	if err := s.RmSink(sink); err != nil {
		return base.NewHyFunErr("RemoveStreamSink", err)
	}
	return nil
}

func (s *streamHandler) Event() []event.HyEvent {
	return []event.HyEvent{
		event.CreateSourceSession,
		event.RemoveSourceSession,
	}
}

func (s *streamHandler) Handle(e event.HyEvent, data *model.EventWrap) {
	log.Infof(data.Ctx, "handle event %d, data: %+v", e, data.Data)
	switch e {
	case event.CreateSourceSession:
		m := data.Data.(map[base.SessionKey]interface{})
		baseI := m[base.ConfigKeyStreamBase].(base.StreamBaseI)
		sess := m[base.ConfigKeyStreamSess].(session.HySessionI)
		stream := NewHyStream(baseI, sess)
		DefaultHyStreamManager.AddStream(stream)
	case event.CreateSinkSession:
		break
	case event.RemoveSourceSession:
		DefaultHyStreamManager.RemoveStream(data.Data.(string))
	case event.RemoveSinkSession:
		break
	default:
		log.Errorf(context.Background(), "event not handle %d %v", e, data)
	}
}
