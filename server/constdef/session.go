package constdef

type SessionType uint16

const (
	SessionTypeInvalid = iota
	SessionTypeRtmpSource

	// SessionSeparator ----------------------
	SessionSeparator
	//SessionSeparator ----------------------

	SessionTypeRtmpSink
	SessionTypeHlsSink
	SessionTypeFileSink
	SessionTypeHttpSink
	SessionTypeInvalid0
)

const DefaultCacheSize = 1024

func (s SessionType) IsSource() bool {
	return s < SessionSeparator
}
