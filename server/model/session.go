package model

import (
	"context"
)

type SinkArg struct {
	Ctx  context.Context
	Sink Sink
}

type Sink interface {
}

type SinkFile struct {
}

type SinkRtmp struct {
}
