package constdef

const (
	Rtmp    = "rtmp"
	HttpFlv = "http-flv"
	HttpApi = "http-api"

	Tcp = "tcp"
	Udp = "udp"
)

type MediaDataType uint16

const (
	_ = iota
	MediaDataTypeVideo
	MediaDataTypeAudio
	MediaDataTypeMetadata
)

func (m MediaDataType) IsVideo() bool {
	return m == MediaDataTypeVideo
}

func (m MediaDataType) IsAudio() bool {
	return m == MediaDataTypeAudio
}

type FrameType byte
type VCodec uint32
type ACodec uint32

const (
	_ FrameType = iota
	Unknown1
	I
	P
	B
	_
)

const (
	_ VCodec = iota
	Unknown2
	H264
	H265
	VMax
)

const (
	_ ACodec = ACodec(VMax)
	Unknown3
	AAC
)
