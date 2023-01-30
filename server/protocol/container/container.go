package container

import "github.com/Opafanls/hylan/server/model"

type ReadWriter interface {
	Init() error
	// Read 格式化packet
	Read(*model.Packet) error
	// Write 写入Packet
	Write(*model.Packet) error
	// OnError error handle
	OnError(err error) (shouldContinue bool)
}
