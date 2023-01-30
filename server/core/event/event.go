package event

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/task"
	"sync"
)

type HyEvent uint32

const (
	invalid             = iota
	CreateSourceSession //新的连接进来
	CreateSinkSession
	RemoveSourceSession
	RemoveSinkSession
	invalid0
)

var DefaultEventHandler0 IEventSrv

func Init() {
	d := &DefaultEventHandler{}
	d.stop = make(chan struct{})
	d.eventHandlers = make(map[HyEvent][]IEventHandler)
	size := 1024
	d.msgChan = make(chan *wrapper, size)
	d.wrapperCache = make(chan *wrapper, size)
	for i := 0; i < size; i++ {
		d.wrapperCache <- &wrapper{data: &model.EventWrap{}}
	}
	DefaultEventHandler0 = d
	DefaultEventHandler0.start()
}

type IEventSrv interface {
	start()
	pushEvent(context.Context, HyEvent, interface{}) error
	handle(w *wrapper)
	register(event HyEvent, handler IEventHandler)
}

type IEventHandler interface {
	Event() []HyEvent
	Handle(event HyEvent, wrapper *model.EventWrap)
}

func RegisterEventHandler(handler IEventHandler) error {
	es := handler.Event()
	for _, e := range es {
		if e <= invalid || e >= invalid0 {
			return fmt.Errorf("event %d invalid", e)
		}
		DefaultEventHandler0.register(e, handler)
	}
	return nil
}

func PushEvent0(ctx context.Context, e HyEvent, data interface{}) error {
	return PushEvent(ctx, e, data, 0)
}

func PushEvent(ctx context.Context, e HyEvent, data interface{}, retry int) error {
	err := DefaultEventHandler0.pushEvent(ctx, e, data)
	if err == nil {
		return nil
	} else {
		if retry == 0 {
			return fmt.Errorf("push failed")
		}
	}
	for i := 0; retry < 0 || i < retry; i++ {
		err = DefaultEventHandler0.pushEvent(ctx, e, data)
		if err == nil {
			break
		}
	}
	return nil
}

//default handler

type DefaultEventHandler struct {
	stop          chan struct{}
	msgChan       chan *wrapper
	wrapperCache  chan *wrapper
	l             sync.Mutex
	eventHandlers map[HyEvent][]IEventHandler
}

type wrapper struct {
	event HyEvent
	data  *model.EventWrap
}

func (d *DefaultEventHandler) pushEvent(ctx context.Context, event HyEvent, data interface{}) error {
	var w *wrapper
	select {
	case w = <-d.wrapperCache:
	default:
		w = &wrapper{data: &model.EventWrap{}}
	}
	w.event = event
	w.data.Ctx = ctx
	w.data.Data = data
	select {
	case d.msgChan <- w:
	default:
		return fmt.Errorf("push failed")
	}
	return nil
}

func (d *DefaultEventHandler) start() {
	task.SubmitTask0(context.Background(), func() {
		for {
			select {
			case <-d.stop:
				return
			case w := <-d.msgChan:
				if w != nil {
					d.handle(w)
				}
			}
		}
	})
}

func (d *DefaultEventHandler) handle(w *wrapper) {
	e := w.event
	hs := d.eventHandlers[e]
	for _, h := range hs {
		h.Handle(w.event, w.data)
	}
}

func (d *DefaultEventHandler) register(event HyEvent, handler IEventHandler) {
	d.l.Lock()
	defer d.l.Unlock()
	d.eventHandlers[event] = append(d.eventHandlers[event], handler)
}
