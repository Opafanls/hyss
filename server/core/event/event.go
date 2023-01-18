package event

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/task"
	"sync"
)

type HyEvent uint32

const (
	invalid       = iota
	CreateSession //新的连接进来
	RemoveSession
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
		d.wrapperCache <- &wrapper{}
	}
	DefaultEventHandler0 = d
	DefaultEventHandler0.start()
}

type IEventSrv interface {
	start()
	pushEvent(HyEvent, interface{}) error
	handle(w *wrapper)
	register(event HyEvent, handler IEventHandler)
}

type IEventHandler interface {
	Event() []HyEvent
	Handle(event HyEvent, data interface{})
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

func PushEvent0(e HyEvent, data interface{}) error {
	return PushEvent(e, data, 0)
}

func PushEvent(e HyEvent, data interface{}, retry int) error {
	err := DefaultEventHandler0.pushEvent(e, data)
	if err == nil {
		return nil
	} else {
		if retry == 0 {
			return fmt.Errorf("push failed")
		}
	}
	for i := 0; retry < 0 || i < retry; i++ {
		err = DefaultEventHandler0.pushEvent(e, data)
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
	data  interface{}
}

func (d *DefaultEventHandler) pushEvent(event HyEvent, data interface{}) error {
	var w *wrapper
	select {
	case w = <-d.wrapperCache:
	default:
		w = &wrapper{}
	}
	w.event = event
	w.data = data
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
