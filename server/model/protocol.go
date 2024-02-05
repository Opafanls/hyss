package model

import (
	"github.com/Opafanls/hylan/server/base"
)

type Packet struct {
	FrameIdx  uint64
	CacheIdx  uint64
	MediaType base.MediaDataType
	Timestamp uint64

	Data   []byte
	Header PacketMeta
}

func (p *Packet) IsKeyFrame() bool {
	if p.IsVideo() {
		vph := p.Header.(*VideoPacketHeader)
		return vph.FrameType == base.I
	}

	return false
}

type PacketMeta interface {
	Tag() string
}

func (p *Packet) IsVideo() bool {
	return p.MediaType == base.MediaDataTypeVideo
}

func (p *Packet) IsMetadata() bool {
	return p.MediaType == base.MediaDataTypeMetadata
}

func (p *Packet) IsAudio() bool {
	return p.MediaType == base.MediaDataTypeAudio
}

type AudioPacketHeader struct {
}

type VideoPacketHeader struct {
	FrameType base.FrameType
	VCodec    base.VCodec
	Tag0      string
}

func NewVideoPktH(frameType base.FrameType, vcodec base.VCodec, tag string) *VideoPacketHeader {
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
	K base.SessionKey
	V interface{}
}

type KVStr struct {
	K string
	V string
}
