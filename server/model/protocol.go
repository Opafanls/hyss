package model

import "github.com/Opafanls/hylan/server/constdef"

type Packet struct {
	FrameIdx  uint64
	MediaType constdef.MediaDataType
	Timestamp uint64

	Data  []byte
	Audio *AudioPacketHeader
	Video *VideoPacketHeader
}

type AudioPacketHeader struct {
}

type VideoPacketHeader struct {
	FrameType constdef.FrameType
	VCodec    constdef.VCodec
}
