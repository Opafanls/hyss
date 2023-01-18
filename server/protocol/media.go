package protocol

type MediaDataType uint16

const (
	_ = iota
	MediaDataTypeVideo
	MediaDataTypeAudio
)

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

type MediaWrapper struct {
	Data          []byte
	FrameType     FrameType
	VCodec        VCodec
	ACodec        ACodec
	MediaDataType MediaDataType
}
