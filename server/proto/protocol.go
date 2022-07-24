package proto

import "github.com/notedit/rtmp/av"

type PacketI interface {
	Raw() []byte
}

type BasePacket struct {
	rawData []byte
}

func (p *BasePacket) Raw() []byte {
	return p.rawData
}

type RtmpPacket struct {
	*BasePacket
	Pkt av.Packet
}
