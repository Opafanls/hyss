package session

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
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
	sessionMap map[int64]HySessionI
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
	hyStreamManager.sessionMap = make(map[int64]HySessionI)
	return hyStreamManager
}

func (streamManager *HyStreamManager) CreateSrcStream(hySession HySessionI) bool {
	if hySession.SessionType().IsSource() {
		return streamManager.addSourceSession(hySession)
	} else {
		return false
	}
}

func (streamManager *HyStreamManager) addSinkSession(hySession HySessionI) bool {
	baseInfo := hySession.Base()
	vhost := baseInfo.Vhost()
	streamName := baseInfo.Name()
	//get stream
	stream := streamManager.GetStream(vhost, streamName)
	if stream == nil {
		return false
	}
	return true
}

func (streamManager *HyStreamManager) addSourceSession(sess HySessionI) bool {
	hyStream := NewHyStream(sess)
	srcSession := hyStream.Source()
	vhost := srcSession.Base().Vhost()
	streamName := srcSession.Base().Name()
	srcSession.SetConfig(base.StreamInitSetSourceStream, hyStream)
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
	}
	vhostMap[streamName] = hyStream
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
func (streamManager *HyStreamManager) StreamFilter(filter func(map[string]map[string]HyStreamI)) error {
	if filter == nil {
		//ret all streams
		return fmt.Errorf("filter is nil")
	}
	streamM := util.Copy(streamManager.streamMap)
	filter(streamM.(map[string]map[string]HyStreamI))
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

func (streamManager *HyStreamManager) RemoveStreamSink(sourceVhost, sourceStreamName string, sinkId int64) error {
	s := streamManager.GetStream(sourceVhost, sourceStreamName)
	if s == nil {
		return base.NewHyFunErr("RemoveStreamSink", fmt.Errorf("source not found"))
	}
	s.RmSink(sinkId)
	return nil
}

func (streamManager *HyStreamManager) Publish(hy HySessionI) error {
	ctx := hy.Ctx()
	info := hy.Base()
	var err = event.PushEvent0(ctx, event.OnSessionCreate, NewStreamEvent(info, hy))
	if err != nil {
		return err
	}
	return hy.Handler().OnPublish(hy.Ctx(), &SourceArg{})
}

func (streamManager *HyStreamManager) Sink(hy HySessionI) error {
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
		sess := m[base.ConfigKeyStreamSess].(HySessionI)
		streamRes := DefaultHyStreamManager.CreateSrcStream(sess)
		log.Infof(sess.Ctx(), "DefaultHyStreamManager.CreateStream with %v", streamRes)
	case event.OnSessionDelete:
		m := data.Data.(map[base.SessionKey]interface{})
		sess := m[base.ConfigKeyStreamSess].(HySessionI)
		DefaultHyStreamManager.DeleteStream(sess.Base().ID())
		log.Infof(sess.Ctx(), "DefaultHyStreamManager.DeleteStream with %v", sess.Base())
	default:
		log.Errorf(context.Background(), "event not handle %d %v", e, data)
	}
}
