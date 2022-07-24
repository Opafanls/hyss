package session

import (
	"context"
	"github.com/Opafanls/hylan/server/proto"
	"github.com/Opafanls/hylan/server/protocol"
)

type HySessionI interface {
	Cycle()
	Data(pkt *proto.PacketI)
	AddSink(arg *proto.SinkArg) HySessionI
}

type HySession struct {
	sessCtx         context.Context
	protocolSession protocol.Protocol
}

func NewHySession(ctx context.Context, ps protocol.Protocol) HySessionI {
	hySession := &HySession{ctx, ps}
	return hySession
}

func (hy *HySession) Cycle() {

}

func (hy *HySession) AddSink(arg *proto.SinkArg) HySessionI {
	return nil
}

func (hy *HySession) Data(pkt *proto.PacketI) {
	return
}
