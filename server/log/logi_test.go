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
	Tracef(context.Background(), "Tracef: name:%s, err: %+v, data: %+v", name, err, data)
	Debugf(context.Background(), "Debugf: name:%s, err: %+v, data: %+v", name, err, data)
	Infof(context.Background(), "Infof: name:%s, err: %+v, data: %+v", name, err, data)
	Warnf(context.Background(), "Warnf: name:%s, err: %+v, data: %+v", name, err, data)
	Errorf(context.Background(), "Errorf: name:%s, err: %+v, data: %+v", name, err, data)
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
	logger.Tracef(context.Background(), "Tracef: name:%s, err: %+v, data: %+v", name, err, data)
	logger.Debugf(context.Background(), "Debugf: name:%s, err: %+v, data: %+v", name, err, data)
	logger.Infof(context.Background(), "Infof: name:%s, err: %+v, data: %+v", name, err, data)
	logger.Warnf(context.Background(), "Warnf: name:%s, err: %+v, data: %+v", name, err, data)
	logger.Errorf(context.Background(), "Errorf: name:%s, err: %+v, data: %+v", name, err, data)
}
