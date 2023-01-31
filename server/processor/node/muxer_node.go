package node

import (
	"context"
	"github.com/Opafanls/hylan/server/core/pipeline"
	"github.com/Opafanls/hylan/server/model"
	"reflect"
)

type MuxerNode struct {
	*pipeline.BaseNode
}

func (m *MuxerNode) Init()          {}
func (m *MuxerNode) Destroy() error { return nil }
func (m *MuxerNode) Name() string {
	return "muxer"
}

func (m *MuxerNode) Action(ctx context.Context, input interface{}) (output interface{}, err error) {
	nodes := m.ListSinkNodes()
	for _, node := range nodes {
		_, err := node.Action(ctx, input)
		m.OnError(ctx, node, err)
	}
	return nil, nil
}

func (m *MuxerNode) InputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&model.Packet{}),
	}
}

func (m *MuxerNode) OutputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&model.Packet{}),
	}
}

func (m *MuxerNode) PutSinkNode(sinkNode pipeline.INode) error {
	return m.BaseNode.PutSinkNode(sinkNode)
}

func (m *MuxerNode) ListSinkNodes() []pipeline.INode {
	return m.BaseNode.ListSinkNodes()
}
