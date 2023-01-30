package model

import "context"

type EventWrap struct {
	Ctx  context.Context
	Data interface{}
}
