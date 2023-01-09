package stream

import (
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/session"
	"net/url"
)

type HyStreamI interface {
	Base() base.StreamBaseI
	Source() session.HySessionI
}

// HyStream biz stream
type HyStream struct {
	StreamBase    base.StreamBaseI
	SourceSession session.HySessionI
}

func NewHyStream0(uri *url.URL, sourceSession session.HySessionI) *HyStream {
	baseData := base.NewBase0(uri)
	return NewHyStream(baseData, sourceSession)
}

func NewHyStream(streamBase base.StreamBaseI, sourceSession session.HySessionI) *HyStream {
	hyStream := &HyStream{}
	hyStream.StreamBase = streamBase
	hyStream.SourceSession = sourceSession
	return hyStream
}

func (stream *HyStream) Base() base.StreamBaseI {
	return stream.StreamBase
}

func (stream *HyStream) Source() session.HySessionI {
	return stream.SourceSession
}
