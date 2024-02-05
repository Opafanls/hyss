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

var DefaultHyStreamManager *HyStreamManager

type HyStreamManagerI interface {
	//StreamFilter(func(map[string]HyStreamI)) error
	//AddStream(hyStream HyStreamI)
	//RemoveStream(streamBaseID string)
	//RemoveStreamSink(source, sink string) error
	//GetStreamByID(id string) HyStreamI
}

type HyStreamManager struct {
	rwLock     *sync.RWMutex
	streamMap  map[string]map[string]HyStreamI //vhost -> stream_name
	sessionMap map[int64]session.HySessionI
}

type EventHandler struct {
}

func InitHyStreamManager() {
	DefaultHyStreamManager = NewHyStreamManager()
	err := event.RegisterEventHandler(&EventHandler{})
	if err != nil {
		log.Fatalf(context.Background(), "register event handler err: %+v", err)
	}
}

func NewHyStreamManager() *HyStreamManager {
	hyStreamManager := &HyStreamManager{}
	hyStreamManager.rwLock = &sync.RWMutex{}
	hyStreamManager.streamMap = make(map[string]map[string]HyStreamI)
	hyStreamManager.sessionMap = make(map[int64]session.HySessionI)
	return hyStreamManager
}

func (streamManager *HyStreamManager) CreateStream(hySession session.HySessionI) bool {
	if hySession.SessionType().IsSource() {
		return streamManager.addSourceSession(hySession)
	} else {
		return streamManager.addSinkSession(hySession)
	}
}

func (streamManager *HyStreamManager) addSinkSession(hySession session.HySessionI) bool {
	baseInfo := hySession.Base()
	vhost := baseInfo.Vhost()
	streamName := baseInfo.Name()
	//get stream
	stream := streamManager.GetStream(vhost, streamName)
	if stream == nil {
		return false
	}
	stream.AddSink(hySession)
	return true
}

func (streamManager *HyStreamManager) addSourceSession(sess session.HySessionI) bool {
	hyStream := NewHyStream(sess)
	vhost := hyStream.Base().Vhost()
	streamName := hyStream.Base().Name()
	srcSession := hyStream.Source()
	streamManager.rwLock.Lock()
	vhostMap := streamManager.streamMap[vhost]
	if vhostMap == nil {
		vhostMap = make(map[string]HyStreamI)
		streamManager.streamMap[vhost] = vhostMap
	} else {
		_, exist := vhostMap[streamName]
		if exist {
			return exist
		}
		vhostMap[streamName] = hyStream
	}
	streamManager.sessionMap[srcSession.Base().ID()] = srcSession
	streamManager.rwLock.Unlock()
	return false
}

func (streamManager *HyStreamManager) DeleteStream(id int64) {
	streamManager.rwLock.Lock()
	defer streamManager.rwLock.Unlock()
	rmSession := streamManager.sessionMap[id]
	if rmSession == nil {
		return
	}
	delete(streamManager.sessionMap, id)
	if rmSession.SessionType().IsSink() {
		return
	}
	vhost := rmSession.Base().Vhost()
	streamName := rmSession.Base().Name()
	vhostMap := streamManager.streamMap[vhost]
	if vhostMap == nil {
		return
	}
	delete(vhostMap, streamName)
	return
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

func (streamManager *HyStreamManager) GetStream(vhost, streamName string) HyStreamI {
	streamManager.rwLock.RLock()
	vhostM := streamManager.streamMap[vhost]
	if vhostM == nil {
		return nil
	}
	result := vhostM[streamName]
	streamManager.rwLock.RUnlock()
	return result
}

func (streamManager *HyStreamManager) RemoveStreamSink(sourceVhost, sourceStreamName, sinkId string) error {
	s := streamManager.GetStream(sourceVhost, sourceStreamName)
	if s == nil {
		return base.NewHyFunErr("RemoveStreamSink", fmt.Errorf("source not found"))
	}
	if err := s.RmSink(sinkId); err != nil {
		return base.NewHyFunErr("RemoveStreamSink", err)
	}
	return nil
}

func (s *EventHandler) Event() []event.HyEvent {
	return []event.HyEvent{
		event.OnSessionCreate,
		event.OnSessionDelete,
	}
}

func (s *EventHandler) Handle(e event.HyEvent, data *model.EventWrap) {
	log.Infof(data.Ctx, "handle event %d, data: %+v", e, data.Data)
	switch e {
	case event.OnSessionCreate:
		m := data.Data.(map[base.SessionKey]interface{})
		sess := m[base.ConfigKeyStreamSess].(session.HySessionI)
		log.Infof(sess.Ctx(), "DefaultHyStreamManager.CreateStream with %v", DefaultHyStreamManager.CreateStream(sess))
	case event.OnSessionDelete:
		m := data.Data.(map[base.SessionKey]interface{})
		sess := m[base.ConfigKeyStreamSess].(session.HySessionI)
		DefaultHyStreamManager.DeleteStream(sess.Base().ID())
		log.Infof(sess.Ctx(), "DefaultHyStreamManager.DeleteStream with %v")
	default:
		log.Errorf(context.Background(), "event not handle %d %v", e, data)
	}
}
