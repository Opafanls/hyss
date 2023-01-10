package task

import (
	"context"
	"github.com/Opafanls/hylan/server/log"
	"github.com/pkg/errors"
	"sync/atomic"
)

var defaultTaskSystem ISystem

func InitTaskSystem() {
	var num int32
	defaultTaskSystem = &DefaultTaskSystem{goroutineNum: &num}
}

func SubmitTask0(ctx context.Context, job Task) {
	err := defaultTaskSystem.SubmitTask(ctx, job)
	if err != nil {
		log.Errorf(ctx, "submit task failed: %+v", err)
		return
	}
}

type Task func()

type ISystem interface {
	SubmitTask(ctx context.Context, job Task) error
	Shutdown()
}

type DefaultTaskSystem struct {
	goroutineNum *int32
}

func (d *DefaultTaskSystem) SubmitTask(ctx context.Context, job Task) error {
	atomic.AddInt32(d.goroutineNum, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errTmp, ok := r.(error)
				if !ok {
					errTmp = errors.Errorf("Panic: %+v", r)
				}
				err := errors.WithStack(errTmp)
				log.Errorf(ctx, "task panic: %+v", err)
			}
		}()
		job()
		atomic.AddInt32(d.goroutineNum, -1)
	}()
	return nil
}

func (d *DefaultTaskSystem) Shutdown() {
	panic("shutdown")
}
