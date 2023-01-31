package pipeline

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type IPipeline interface {
	Print() string
	Check() (errIndex int, err error)
	Fire(ctx context.Context, input interface{}) error
	Stop() error
	FirstNode() (node INode, exist bool)
	LastNode() (node INode, exist bool)
	MountFirst(node INode, override bool) error
	//todo get pipeline run result
}

type PipeCtx interface {
	context.Context
	Result(sync bool) interface{}
}

type DefaultPipeline struct {
	firstNode INode
	lastNode  INode

	running bool
	l       sync.Mutex
}

func NewDefaultP() *DefaultPipeline {
	de := &DefaultPipeline{}
	de.running = true
	return de
}

func (d *DefaultPipeline) Print() string {
	return "\n" + d.printNode(d.firstNode)
}

func (d *DefaultPipeline) printNode(node INode) string {
	name := node.Name()
	input := node.InputType()
	output := node.OutputType()
	curPrint := fmt.Sprintf("%s[in:%v][out:%v]\t", name, input, output)
	for _, sink := range node.ListSinkNodes() {
		sinkPrint := d.printNode(sink)
		curPrint = fmt.Sprintf("%s\n\t%s", curPrint, sinkPrint)
	}

	return curPrint
}

func (d *DefaultPipeline) Check() (errIndex int, err error) {
	if d.firstNode == nil {
		return 0, fmt.Errorf("firstNode is nil")
	}
	return d.check(0, d.firstNode)
}

func (d *DefaultPipeline) check(idx int, curNode INode) (int, error) {
	sinkNodes := curNode.ListSinkNodes()
	for _, sinkNode := range sinkNodes {
		inputTypes := sinkNode.InputType()
		outputTypes := curNode.OutputType()
		if len(outputTypes) == 0 {
			return idx, fmt.Errorf("node output type is empty")
		}
		for _, output := range outputTypes {
			if !d.typeIn(inputTypes, output) {
				return idx, fmt.Errorf("%s->%s input type %+v can not handle output type: %+v", curNode.Name(), sinkNode.Name(), inputTypes, output)
			}
		}
		if errIdx, err := d.check(idx+1, sinkNode); err != nil {
			return errIdx, err
		}
	}
	return 0, nil
}

func (d *DefaultPipeline) typeIn(ins []reflect.Type, target reflect.Type) bool {
	for _, t := range ins {
		if t == target {
			return true
		}
	}
	return false
}

func (d *DefaultPipeline) Fire(ctx context.Context, input interface{}) error {
	if d.firstNode == nil {
		return fmt.Errorf("pipeline first node is nil")
	}
	return d.fire(ctx, d.firstNode, input)
}

func (d *DefaultPipeline) fire(ctx context.Context, node INode, input interface{}) error {
	_, err := node.Action(ctx, input)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultPipeline) Stop() error {
	d.l.Lock()
	d.running = false
	d.l.Unlock()
	return nil
}

func (d *DefaultPipeline) FirstNode() (node INode, exist bool) {
	return d.firstNode, d.firstNode != nil
}

func (d *DefaultPipeline) LastNode() (node INode, exist bool) {
	return d.lastNode, d.lastNode != nil
}

func (d *DefaultPipeline) MountFirst(node INode, override bool) error {
	if override {
		d.firstNode = node
		return nil
	}
	if d.firstNode != nil {
		return ErrFirstNodeExist
	}
	d.firstNode = node
	return nil
}

//func (d *DefaultPipeline) MountLast(node INode) error {
//	if d.lastNode == nil {
//		d.firstNode, d.lastNode = node, node
//		return nil
//	}
//
//}
