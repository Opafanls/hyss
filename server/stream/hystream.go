package stream

import (
	"encoding/json"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/session"
	"sync"
)

type HyStreamI interface {
	Base() base.StreamBaseI
	Source() session.HySessionI
	Sink(arg *session.SinkArg) error
	RmSink(string) error
	AddSink(session.HySessionI)
}

// HyStream biz stream
type HyStream struct {
	sourceSession session.HySessionI
	sinkSessions  map[int64]session.HySessionI
	l             sync.Mutex
}

func NewHyStream0(sourceSession session.HySessionI) *HyStream {
	return NewHyStream(sourceSession)
}

func NewHyStream(sourceSession session.HySessionI) *HyStream {
	hyStream := &HyStream{}
	hyStream.sourceSession = sourceSession
	hyStream.sinkSessions = make(map[int64]session.HySessionI)
	return hyStream
}

func (stream *HyStream) Base() base.StreamBaseI {
	return stream.sourceSession.Base()
}

func (stream *HyStream) Source() session.HySessionI {
	return stream.sourceSession
}

func (stream *HyStream) AddSink(s session.HySessionI) {
	stream.l.Lock()
	stream.sinkSessions[s.Base().ID()] = s
	stream.l.Unlock()
}

func (stream *HyStream) Sink(arg *session.SinkArg) error {
	return nil
}

func (stream *HyStream) RmSink(id string) error {
	return nil
}

func (stream *HyStream) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 2)
	m["stream_source"] = stream.sourceSession
	m["stream_sinks"] = stream.sinkSessions
	return json.Marshal(m)
}
