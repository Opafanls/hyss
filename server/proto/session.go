package proto

import "github.com/Opafanls/hylan/server/constdef"

type SinkArg struct {
	Protocol constdef.SinkType
	SinkFile *SinkFile
	SinkRtmp *SinkRtmp
}

type SinkFile struct {
}

type SinkRtmp struct {
}
