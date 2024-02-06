package protocol

import (
	"github.com/Opafanls/hylan/server/core/av"
)

type PacketReceiver struct {
	ch        chan *av.Packet
	runningCh chan struct{}
	running   bool
}

type OnPacketHandler func(pkt *av.Packet)

func NewPacketReceiver(handler OnPacketHandler) *PacketReceiver {
	s := &PacketReceiver{}
	s.Init(handler)
	return s
}
func (p *PacketReceiver) Init(handler OnPacketHandler) {
	p.ch = make(chan *av.Packet, 128)
	p.runningCh = make(chan struct{}, 0)
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
	select {
	case <-p.runningCh:
		if p.running {
			p.running = false
			close(p.ch)
			return
		}
	default:
		p.ch <- pkt
	}
}

func (p *PacketReceiver) Pull() *av.Packet {
	return <-p.ch
}

func (p *PacketReceiver) Stop() {
	close(p.runningCh)
}
