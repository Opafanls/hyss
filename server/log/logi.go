package log

import (
	"context"
	"time"
)

var MainLog ILog
var Ctx = context.Background()

func init() {
	MainLog = &Logrus{}
	MainLog.Init(nil)
}

type ILog interface {
	Init(param interface{}) error
	Tracef(ctx context.Context, format string, args ...interface{})
	Debugf(ctx context.Context, format string, args ...interface{})
	Infof(ctx context.Context, format string, args ...interface{})
	Warnf(ctx context.Context, format string, args ...interface{})
	Errorf(ctx context.Context, format string, args ...interface{})
}

func Tracef(ctx context.Context, format string, args ...interface{}) {
	MainLog.Tracef(ctx, format, args...)
}
func Debugf(ctx context.Context, format string, args ...interface{}) {
	MainLog.Debugf(ctx, format, args...)
}
func Infof(ctx context.Context, format string, args ...interface{}) {
	MainLog.Infof(ctx, format, args...)
}
func Warnf(ctx context.Context, format string, args ...interface{}) {
	MainLog.Warnf(ctx, format, args...)
}
func Errorf(ctx context.Context, format string, args ...interface{}) {
	MainLog.Errorf(ctx, format, args...)
}

func GetEmptyCtx() context.Context {
	return context.Background()
}

func GetCtxWithLogID() context.Context {
	ctx := context.WithValue(context.Background(), "__LOGID__", time.Now().UnixNano())
	return ctx
}
