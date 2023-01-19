package session

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
)

type SinkArg struct {
	Ctx    context.Context
	Sink   Sink
	Remote HySessionI
	Local  HySessionI
}

type Sink interface {
	Base() base.StreamBaseI
	Type() string
}

type BaseSink struct {
	base  base.StreamBaseI
	ttype string
}

func (base *BaseSink) Base() base.StreamBaseI {
	return base.base
}

func (base *BaseSink) Type() string {
	return base.ttype
}

func NewBaseSink(base base.StreamBaseI, ttype string) *BaseSink {
	return &BaseSink{
		base:  base,
		ttype: ttype,
	}
}

type SinkFile struct {
	*BaseSink
}

type SinkRtmp struct {
	*BaseSink
}
