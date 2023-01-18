package log

import (
	"context"
	"fmt"
	"time"
)

var MainLog ILog

const LOG_KEY = "_HY_LOGID_"

var EMPTY_TAGS []string

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
	Fatalf(ctx context.Context, format string, args ...interface{})
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

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	MainLog.Fatalf(ctx, format, args...)
}

func GetEmptyCtx() context.Context {
	return context.Background()
}

func generateLogID(tag string) []string {
	var tags []string
	timeNow := time.Now()
	timestamp := timeNow.Format("20060102150405")
	if tag != "" {
		tags = append(tags, tag)
	}
	timestamp = fmt.Sprintf("%s%d", timestamp, timeNow.UnixNano())
	tags = append(tags, timestamp)
	return tags
}

func getLogIDFromCtx(ctx context.Context) []string {
	if ctx == nil {
		return EMPTY_TAGS
	}
	values, ok := ctx.Value(LOG_KEY).([]string)
	if ok {
		return values
	}
	return EMPTY_TAGS
}

func getLogIDFormat(ctx context.Context, oldFormat string) string {
	tags := getLogIDFromCtx(ctx)
	if len(tags) > 0 {
		return fmt.Sprintf("%+v %s", tags, oldFormat)
	}
	return oldFormat
}

func GetCtxWithLogID(parent context.Context, tag string) context.Context {
	ctx := context.WithValue(parent, LOG_KEY, generateLogID(tag))
	return ctx
}

func SetLogIDToCtx(parentCtx context.Context, logTags []string) context.Context {
	return context.WithValue(parentCtx, LOG_KEY, logTags)
}
