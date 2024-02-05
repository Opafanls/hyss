package base

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

func NewHyFunErr(funcName string, errMsg error) *HyError {
	return &HyError{
		CtxMsg: funcName,
		Err:    errMsg,
	}
}

func (h *HyError) Error() string {
	if h.Err == nil {
		return fmt.Sprintf("Err:no_err;Msg:%s", h.CtxMsg)
	}
	return fmt.Sprintf("Err:%+v;Msg:%s", h.Err, h.CtxMsg)
}

var (
	SessionCannotPushMedia = NewHyErrorWithSimpleMsg(fmt.Errorf("session can not push media"))
	SessionCannotPullMedia = NewHyErrorWithSimpleMsg(fmt.Errorf("session can not pull media"))
)
