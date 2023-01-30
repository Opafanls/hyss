package model

import "github.com/Opafanls/hylan/server/constdef"

type Packet struct {
	FrameIdx  uint64
	MediaType constdef.MediaDataType
	Timestamp uint64

	Data   []byte
	Header PacketMeta
}

func (p *Packet) IsKeyFrame() bool {
	if p.IsVideo() {
		vph := p.Header.(*VideoPacketHeader)
		return vph.FrameType == constdef.I
	}

	return false
}

type PacketMeta interface {
	Tag() string
}

func (p *Packet) IsVideo() bool {
	return p.MediaType == constdef.MediaDataTypeVideo
}

func (p *Packet) IsMetadata() bool {
	return p.MediaType == constdef.MediaDataTypeMetadata
}

func (p *Packet) IsAudio() bool {
	return p.MediaType == constdef.MediaDataTypeAudio
}

type AudioPacketHeader struct {
}

type VideoPacketHeader struct {
	FrameType constdef.FrameType
	VCodec    constdef.VCodec
	Tag0      string
}

func NewVideoPktH(frameType constdef.FrameType, vcodec constdef.VCodec, tag string) *VideoPacketHeader {
	v := &VideoPacketHeader{
		FrameType: frameType,
		VCodec:    vcodec,
		Tag0:      tag,
	}
	return v
}

func (v *VideoPacketHeader) Tag() string {
	return v.Tag0
}

type KV struct {
	K constdef.SessionConfigKey
	V interface{}
}

type KVStr struct {
	K string
	V string
}
