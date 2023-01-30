package rtmp

import (
	"fmt"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol/container"
	"github.com/Opafanls/hylan/server/session"
	"time"
)

type Rtmp struct {
	*container.RWBaser
	session session.HySessionI
	h       *RtmpHandler
	cs      *chunkStream
}

func NewRtmp(h *RtmpHandler) *Rtmp {
	r := &Rtmp{}
	r.RWBaser = container.NewRWBaser(time.Second * 10)
	r.h = h
	r.cs = newChunkStream(h.Ctx, h, 0, 0)
	return r
}

func (r *Rtmp) Init() error {
	return nil
}

func (r *Rtmp) Read(packet *model.Packet) error {
	if packet.Header.Tag() == "rtmp" {
		return nil
	}
	return nil
}

func (r *Rtmp) Write(p *model.Packet) error {
	//write packet to conn
	tag := p.Header.Tag()
	cs := r.cs
	cs.writeBuffer.Reset()
	rtmpMeta, ok := (p.Header).(*RtmpPacketMeta)
	if !ok {
		return fmt.Errorf("packet meta type case failed")
	}
	if tag == "rtmp" {
		//write direct
		cs.writeBuffer.Write(p.Data)
		cs.msgLen = uint32(len(p.Data))
		cs.streamID = rtmpMeta.StreamID
		cs.timestamp = uint32(p.Timestamp)
		cs.timestamp += r.BaseTimeStamp()
		var ty uint32
		if p.IsVideo() {
			cs.typeID = TypeIDVideoMessage
			ty = container.Video
		} else {
			if p.IsMetadata() {
				cs.typeID = TypeIDDataMessageAMF0
			} else {
				cs.typeID = TypeIDAudioMessage
				ty = container.Audio
			}
		}
		r.SetPreTime()
		r.RecTimeStamp(cs.timestamp, ty)
		err := cs.writeChunk(&chunkPayload{
			chunkHeader: cs.chunkHeader,
			chunkData:   p.Data,
		})
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("write packet %s not valid now", tag)
}
