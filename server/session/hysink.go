package session

import (
	"context"
)

type SourceArg struct {
}

type SinkArg struct {
	Ctx         context.Context
	Sink        Sink
	SinkSession HySessionI
}

type Sink interface {
	Type() string
}

type SinkFile struct {
}

func (sk *SinkFile) Type() string {
	return "file"
}

type SinkRtmp struct {
}

func (sr *SinkRtmp) Type() string {
	return "rtmp"
}
