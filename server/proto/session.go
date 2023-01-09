package proto

import (
	"context"
	"github.com/Opafanls/hylan/server/constdef"
)

type SinkArg struct {
	Ctx      context.Context
	Protocol constdef.SinkType
	SinkFile *SinkFile
	SinkRtmp *SinkRtmp
}

type SinkFile struct {
}

type SinkRtmp struct {
}
