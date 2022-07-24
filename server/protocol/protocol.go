package protocol

import (
	"context"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/proto"
)

type ListenArg struct {
	Ip   string
	Port int
}

type Protocol interface {
	Name() string
	Listen(arg *ListenArg)
	SetData(packet proto.PacketI) *constdef.HyError
	GetData() (proto.PacketI, *constdef.HyError)
	Stop(msg string) *constdef.HyError
}

type BaseProtocol struct {
	ctx context.Context
}

func (b *BaseProtocol) Name() string {
	//TODO implement me
	panic("implement me")
}

func (b *BaseProtocol) Listen(arg *ListenArg) {
	//TODO implement me
	panic("implement me")
}

func (b *BaseProtocol) SetData(packet proto.PacketI) *constdef.HyError {
	//TODO implement me
	panic("implement me")
}

func (b *BaseProtocol) GetData() (*proto.BasePacket, *constdef.HyError) {
	//TODO implement me
	panic("implement me")
}

func (b BaseProtocol) Stop(msg string) *constdef.HyError {
	//TODO implement me
	panic("implement me")
}
