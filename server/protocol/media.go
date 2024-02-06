package protocol

import (
	"github.com/Opafanls/hylan/server/core/av"
	"github.com/Opafanls/hylan/server/log"
)

type PacketReceiver struct {
	ch      chan *av.Packet
	running bool
}

type OnPacketHandler func(pkt *av.Packet)

func NewPacketReceiver(handler OnPacketHandler) *PacketReceiver {
	s := &PacketReceiver{}
	s.Init(handler)
	return s
}
func (p *PacketReceiver) Init(handler OnPacketHandler) {
	p.ch = make(chan *av.Packet, 128)
	p.running = true
	go func() {
		for p.running {
			select {
			case pkt := <-p.ch:
				if handler != nil {
					handler(pkt)
				}
			}
		}
	}()
}

func (p *PacketReceiver) Push(pkt *av.Packet) {
	if !p.running {
		return
	}
	select {
	case p.ch <- pkt:
	default:
		log.Infof(nil, "empty push...")
	}
}

func (p *PacketReceiver) Pull() *av.Packet {
	return <-p.ch
}

func (p *PacketReceiver) Stop() {
	p.running = false
	close(p.ch)
}
