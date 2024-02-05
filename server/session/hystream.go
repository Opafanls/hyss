package session

import (
	"encoding/json"
	"github.com/Opafanls/hylan/server/base"
	"sync"
)

type HyStreamI interface {
	Base() base.StreamBaseI
	Source() HySessionI
	Sink(arg *SinkArg) error
	RmSink(string) error
	AddSink(HySessionI)
}

// HyStream biz stream
type HyStream struct {
	sourceSession HySessionI
	sinkSessions  map[int64]HySessionI
	l             sync.Mutex
}

func NewHyStream0(sourceSession HySessionI) *HyStream {
	return NewHyStream(sourceSession)
}

func NewHyStream(sourceSession HySessionI) *HyStream {
	hyStream := &HyStream{}
	hyStream.sourceSession = sourceSession
	hyStream.sinkSessions = make(map[int64]HySessionI)
	return hyStream
}

func (stream *HyStream) Base() base.StreamBaseI {
	return stream.sourceSession.Base()
}

func (stream *HyStream) Source() HySessionI {
	return stream.sourceSession
}

func (stream *HyStream) AddSink(s HySessionI) {
	stream.l.Lock()
	stream.sinkSessions[s.Base().ID()] = s
	stream.l.Unlock()
}

func (stream *HyStream) Sink(arg *SinkArg) error {
	stream.AddSink(arg.SinkSession)
	err := stream.Source().Handler().OnSink(arg.Ctx, arg)
	if err != nil {
		return err
	}
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