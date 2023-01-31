package pipeline

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type NodeEvent int

const (
	_ NodeEvent = iota
)

type INode interface {
	Name() string                                                                  //node名称
	Action(ctx context.Context, input interface{}) (output interface{}, err error) //处理器
	InputType() []reflect.Type                                                     //输入的数据类型
	OutputType() []reflect.Type                                                    //输出的数据类型
	PutSinkNode(sinkNode INode) error                                              //增加数据输出node
	ListSinkNodes() []INode
	INodeEvent    //事件模型
	NodeLifecycle //生命周期
}

type NodeLifecycle interface {
	Init()
	Destroy() error
}

type INodeEvent interface {
}

type ErrorHandler func(ctx context.Context, node INode, err error)

func NewBaseNode() *BaseNode {
	return &BaseNode{}
}

type BaseNode struct {
	sinkNodes  []INode
	l          sync.Mutex
	ErrHandler ErrorHandler
}

func (b *BaseNode) Init() {}

func (b *BaseNode) Destroy() error { return nil }

func (b *BaseNode) Name() string {
	return "base"
}

func (b *BaseNode) Action(ctx context.Context, input interface{}) (output interface{}, err error) {
	return nil, nil
}

func (b *BaseNode) InputType() []reflect.Type {
	return []reflect.Type{}
}

func (b *BaseNode) OutputType() []reflect.Type {
	return []reflect.Type{}
}

func (b *BaseNode) PutSinkNode(sinkNode INode) error {
	b.l.Lock()
	defer b.l.Unlock()
	for _, in0 := range b.OutputType() {
		var exist = true
		for _, in := range sinkNode.InputType() {
			if in0 == in {
				exist = true
			}
		}
		if !exist {
			return fmt.Errorf("node output type not handle for %s->%s", b.Name(), sinkNode.Name())
		}
	}

	b.sinkNodes = append(b.sinkNodes, sinkNode)
	return nil
}

func (b *BaseNode) ListSinkNodes() []INode {
	return b.sinkNodes
}

func (b *BaseNode) OnError(ctx context.Context, node INode, err error) {
	if b.ErrHandler != nil {
		b.ErrHandler(ctx, node, err)
	}
}
