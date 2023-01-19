package stream

import (
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/session"
	"net/url"
	"sync"
	"time"
)

type HyStreamI interface {
	Base() base.StreamBaseI
	Source() session.HySessionI
	Sink(arg *session.SinkArg) error
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
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	stream.l.Lock()
	stream.sinkSessions[id] = arg.Remote
	stream.l.Unlock()
	defer func() {
		stream.l.Lock()
		delete(stream.sinkSessions, id)
		stream.l.Unlock()
	}()
	err := stream.sourceSession.Sink(arg)
	if err != nil {
		return err
	}
	return nil
}

func (stream *HyStream) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 2)
	m["stream_info"] = stream.base
	m["stream_source"] = stream.sourceSession
	m["stream_sinks"] = stream.sinkSessions
	return json.Marshal(m)
}
