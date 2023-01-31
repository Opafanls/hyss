package node

import (
	"context"
	"github.com/Opafanls/hylan/server/core/pipeline"
	"github.com/Opafanls/hylan/server/model"
	"reflect"
	"time"
)

//async process input node

type AsyncNode struct {
	*pipeline.BaseNode
	ch        chan interface{}
	sl        time.Duration
	handleFun func(data interface{})
	stop      chan struct{}
}

func (m *AsyncNode) Init() {
	go func() {
		for {
			select {
			case d, ok := <-m.ch:
				if ok {
					m.handleFun(d)
				}
			case <-m.stop:
				return
			}
		}
	}()
}

func (m *AsyncNode) Destroy() error {
	close(m.stop)
	close(m.ch)
	return nil
}

func (m *AsyncNode) Name() string {
	return "async"
}

func (m *AsyncNode) Action(ctx context.Context, input interface{}) (output interface{}, err error) {
	if m.sl == 0 {
		//block
		m.ch <- input
	} else {
		for {
			select {
			case m.ch <- input:
				return nil, nil
			default:
				time.Sleep(m.sl)
			}
		}
	}
	return nil, nil
}

func (m *AsyncNode) InputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&model.Packet{}),
	}
}

func (m *AsyncNode) OutputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&model.Packet{}),
	}
}
