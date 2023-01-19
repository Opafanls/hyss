package constdef

type SessionType uint16

const (
	SessionTypeInvalid = iota
	SessionTypeSource
	SessionTypeSink
	SessionTypeInvalid0
)

const DefaultCacheSize = 1024
