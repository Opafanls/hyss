package flv

import (
	"fmt"
	"github.com/Opafanls/hylan/server/core/bitio"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol/container"
	"github.com/Opafanls/hylan/server/protocol/data_format/amf"
	"io"
	"time"
)

var (
	flvHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}
	pad       = []byte{0, 0, 0, 0}
	headerLen = 11
)

type Flv struct {
	*Muxer
	*Demuxer
}

func NewFlv(ctx io.WriteCloser) *Flv {
	f := &Flv{
		Muxer:   newFLVWriter(ctx),
		Demuxer: newDemuxer(),
	}
	return f
}

func (f *Flv) Init() error {
	if f.ctx != nil {
		return f.Muxer.Init()
	}
	return nil
}

func (f *Flv) Read(packet *model.Packet) error {
	return f.Demuxer.Demux(packet)
}

func (f *Flv) Write(packet *model.Packet) error {
	return f.Muxer.Write(packet)
}

type Muxer struct {
	*container.RWBaser
	//app, title, url string
	buf []byte
	ctx io.WriteCloser
}

func newFLVWriter(ctx io.WriteCloser) *Muxer {
	ret := &Muxer{
		//title:   title,
		//url:     url,
		ctx:     ctx,
		RWBaser: container.NewRWBaser(time.Second * 10),
		buf:     make([]byte, headerLen),
	}
	return ret
}

func (writer *Muxer) Init() error {
	if _, err := writer.ctx.Write(flvHeader); err != nil {
		return err
	}
	if _, err := writer.ctx.Write(pad); err != nil {
		return err
	}
	return nil
}

func (writer *Muxer) Write(p *model.Packet) error {
	writer.RWBaser.SetPreTime()
	h := writer.buf[:headerLen]
	typeID := Tag_Video
	if !p.IsVideo() {
		if p.IsMetadata() {
			var err error
			typeID = Tag_ScriptDataAMF3
			p.Data, err = amf.MetaDataReform(p.Data, amf.DEL)
			if err != nil {
				return err
			}
		} else {
			typeID = Tag_Audio
		}
	}
	dataLen := len(p.Data)
	timestamp := p.Timestamp
	timestamp += uint64(writer.BaseTimeStamp())
	var t uint32 = 0
	if typeID == Tag_Video {
		t = container.Video
	} else if typeID == Tag_Audio {
		t = container.Audio
	}
	writer.RWBaser.RecTimeStamp(uint32(timestamp), t)

	preDataLen := dataLen + headerLen
	timestampbase := timestamp & 0xffffff
	timestampExt := timestamp >> 24 & 0xff

	bitio.PutU8(h[0:1], uint8(typeID))
	bitio.PutI24BE(h[1:4], int32(dataLen))
	bitio.PutI24BE(h[4:7], int32(timestampbase))
	bitio.PutU8(h[7:8], uint8(timestampExt))

	if _, err := writer.ctx.Write(h); err != nil {
		return err
	}

	if _, err := writer.ctx.Write(p.Data); err != nil {
		return err
	}

	bitio.PutI32BE(h[:4], int32(preDataLen))
	if _, err := writer.ctx.Write(h[:4]); err != nil {
		return err
	}

	return nil
}

var (
	ErrAvcEndSEQ = fmt.Errorf("avc end sequence")
)

type Demuxer struct {
}

func newDemuxer() *Demuxer {
	return &Demuxer{}
}

func (d *Demuxer) DemuxHeader(p *model.Packet) error {
	var tag Tag
	_, err := tag.ParseMediaTagHeader(p.Data, p.IsVideo())
	if err != nil {
		return err
	}
	p.Header = &tag

	return nil
}

func (d *Demuxer) Demux(p *model.Packet) error {
	var tag Tag
	n, err := tag.ParseMediaTagHeader(p.Data, p.IsVideo())
	if err != nil {
		return err
	}
	if tag.CodecID() == H264 &&
		p.Data[0] == 0x17 && p.Data[1] == 0x02 {
		return ErrAvcEndSEQ
	}
	p.Header = &tag
	p.Data = p.Data[n:]
	return nil
}
