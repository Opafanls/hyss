package stream

import (
	"encoding/json"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/event"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/session"
	"net/url"
	"sync"
)

type HyStreamI interface {
	Base() base.StreamBaseI
	Source() session.HySessionI
	Sink(arg *session.SinkArg) error
	RmSink(string) error
}

// HyStream biz stream
type HyStream struct {
	base          base.StreamBaseI
	sourceSession session.HySessionI
	sinkSessions  map[string]session.HySessionI
	l             sync.Mutex
}

func NewHyStream0(uri *url.URL, sourceSession session.HySessionI) *HyStream {
	baseData := base.NewBase0(uri)
	return NewHyStream(baseData, sourceSession)
}

func NewHyStream(streamBase base.StreamBaseI, sourceSession session.HySessionI) *HyStream {
	hyStream := &HyStream{}
	hyStream.base = streamBase
	hyStream.sourceSession = sourceSession
	hyStream.sinkSessions = make(map[string]session.HySessionI)
	return hyStream
}

func (stream *HyStream) Base() base.StreamBaseI {
	return stream.base
}

func (stream *HyStream) Source() session.HySessionI {
	return stream.sourceSession
}

func (stream *HyStream) Sink(arg *session.SinkArg) error {
	stream.l.Lock()
	stream.sinkSessions[arg.SourceID] = arg.Remote
	stream.l.Unlock()
	defer func() {
		stream.l.Lock()
		delete(stream.sinkSessions, arg.SourceID)
		stream.l.Unlock()
		err := event.PushEvent0(arg.Ctx, event.RemoveSinkSession, &model.KVStr{
			K: arg.SourceID,
			V: arg.Local.Base().ID(),
		})
		if err != nil {
			log.Errorf(arg.Ctx, "push RemoveSinkSession err: %+v", err)
		}
	}()
	err := stream.sourceSession.Sink(arg)
	if err != nil {
		return err
	}
	return nil
}

func (stream *HyStream) RmSink(id string) error {
	stream.l.Lock()
	delete(stream.sinkSessions, id)
	stream.l.Unlock()
	return nil
}

func (stream *HyStream) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 2)
	m["stream_info"] = stream.base
	m["stream_source"] = stream.sourceSession
	m["stream_sinks"] = stream.sinkSessions
	return json.Marshal(m)
}
