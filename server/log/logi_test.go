package log

import (
	"context"
	"errors"
	"testing"
)

func TestMainLog(t *testing.T) {
	name := "cnss"
	err := errors.New("test err")
	data := map[string]interface{}{
		"1": 1,
		"2": "2",
	}
	Tracef(context.Background(), "Tracef: name:%s, err: %+v, data_format: %+v", name, err, data)
	Debugf(context.Background(), "Debugf: name:%s, err: %+v, data_format: %+v", name, err, data)
	Infof(context.Background(), "Infof: name:%s, err: %+v, data_format: %+v", name, err, data)
	Warnf(context.Background(), "Warnf: name:%s, err: %+v, data_format: %+v", name, err, data)
	Errorf(context.Background(), "Errorf: name:%s, err: %+v, data_format: %+v", name, err, data)
}

func TestLogrus(t *testing.T) {
	logger := &Logrus{}
	logger.Init(nil)
	name := "cnss"
	err := errors.New("test err")
	data := map[string]interface{}{
		"1": 1,
		"2": "2",
	}
	logger.Tracef(context.Background(), "Tracef: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Debugf(context.Background(), "Debugf: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Infof(context.Background(), "Infof: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Warnf(context.Background(), "Warnf: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Errorf(context.Background(), "Errorf: name:%s, err: %+v, data_format: %+v", name, err, data)
}

func TestLogrusWithCtx(t *testing.T) {
	logger := &Logrus{}
	logger.Init(nil)
	name := "cnss"
	err := errors.New("test err")
	data := map[string]interface{}{
		"1": 1,
		"2": "2",
	}
	ctx := context.Background()
	logger.Tracef(GetCtxWithLogID(ctx, "tag"), "Tracef: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Debugf(GetCtxWithLogID(ctx, "tag"), "Debugf: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Infof(GetCtxWithLogID(ctx, "tag"), "Infof: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Warnf(GetCtxWithLogID(ctx, "tag"), "Warnf: name:%s, err: %+v, data_format: %+v", name, err, data)
	logger.Errorf(GetCtxWithLogID(ctx, "tag"), "Errorf: name:%s, err: %+v, data_format: %+v", name, err, data)
}
