package constdef

type SinkType uint16

type SessionType uint16

const (
	SinkTypeFile SinkType = iota
	SinkTypeRtmp
)

const (
	SessionTypeInvalid = iota
	SessionTypeSource
	SessionTypeSink
	SessionTypeInvalid0
)

const DefaultCacheSize = 1024
