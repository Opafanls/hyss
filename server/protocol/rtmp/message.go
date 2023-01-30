package rtmp

import "github.com/Opafanls/hylan/server/model"

type RtmpPacketMeta struct {
	*model.VideoPacketHeader
	StreamID uint32
}

func NewRtmpPacketMeta(v *model.VideoPacketHeader, streamID uint32) *RtmpPacketMeta {
	return &RtmpPacketMeta{
		VideoPacketHeader: v,
		StreamID:          streamID,
	}
}
