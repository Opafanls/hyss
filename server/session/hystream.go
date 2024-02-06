package session

import (
	"encoding/json"
	"sync"
)

type HyStreamI interface {
	Source() HySessionI                                     //对应的source session
	Sink(arg *SinkArg) error                                //增加一路sink
	RmSink(int64)                                           //删除一路sink
	RangeSinks(rangeFun func(id int64, session HySessionI)) //遍历sink
}

// HyStream biz stream
type HyStream struct {
	sourceSession HySessionI
	sinkSessions  *sync.Map
	l             sync.Mutex
}

func NewHyStream0(sourceSession HySessionI) *HyStream {
	return NewHyStream(sourceSession)
}

func NewHyStream(sourceSession HySessionI) *HyStream {
	hyStream := &HyStream{}
	hyStream.sourceSession = sourceSession
	hyStream.sinkSessions = &sync.Map{}
	return hyStream
}

func (stream *HyStream) Source() HySessionI {
	return stream.sourceSession
}

func (stream *HyStream) addSink(s HySessionI) {
	stream.sinkSessions.Store(s.Base().ID(), s)
}

func (stream *HyStream) Sink(arg *SinkArg) error {
	stream.addSink(arg.SinkSession)
	arg.SourceSession = stream.Source()
	if err := arg.SinkSession.Sink(arg); err != nil {
		return err
	}
	return nil
}

func (stream *HyStream) RmSink(id int64) {
	stream.sinkSessions.Delete(id)
}

func (stream *HyStream) RangeSinks(rangeFun func(id int64, session HySessionI)) {
	stream.sinkSessions.Range(func(id, session any) bool {
		id0 := id.(int64)
		session0 := session.(HySessionI)
		rangeFun(id0, session0)
		return true
	})
}

func (stream *HyStream) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, 2)
	m["stream_source"] = stream.sourceSession
	sinkM := make(map[int64]HySessionI)
	stream.sinkSessions.Range(func(key, value any) bool {
		id := key.(int64)
		session := value.(HySessionI)
		sinkM[id] = session
		return true
	})
	m["stream_sinks"] = sinkM
	return json.Marshal(m)
}
