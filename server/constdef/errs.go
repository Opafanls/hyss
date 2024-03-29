package constdef

import (
	"context"
	"fmt"
)

type HyError struct {
	Err    error
	CtxMsg string
	Ctx    context.Context
}

func NewHyErrorWithSimpleMsg(err error) *HyError {
	return &HyError{
		Err:    err,
		CtxMsg: "",
	}
}

func NewHyError(ctxMsg string, err error) *HyError {
	return &HyError{
		Err: err, CtxMsg: ctxMsg,
	}
}

func (h *HyError) Error() string {
	if h.Err == nil {
		return fmt.Sprintf("Err:no_err;Msg:%s", h.CtxMsg)
	}
	return fmt.Sprintf("Err:%+v;Msg:%s", h.Err, h.CtxMsg)
}
