package container

import "github.com/Opafanls/hylan/server/model"

type ReadWriter interface {
	Read() (*model.Packet, error)
	Write(*model.Packet) error
	OnError(err error) bool
}
