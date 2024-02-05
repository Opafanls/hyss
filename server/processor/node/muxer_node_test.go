package node

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/pipeline"
	"github.com/Opafanls/hylan/server/model"
	"reflect"
	"testing"
)

var mockPipe = &pipeline.DefaultPipeline{}

type item struct {
	fieldInt int
}

type printNode struct {
	idx int
	*pipeline.BaseNode
}

func (p *printNode) Init()          {}
func (p *printNode) Destroy() error { return nil }
func (p *printNode) Action(ctx context.Context, input interface{}) (output interface{}, err error) {
	fmt.Printf("%d action: %v\n", p.idx, input)
	return nil, nil
}

func (p *printNode) InputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&item{}),
		reflect.TypeOf(&model.Packet{}),
	}
}
func (p *printNode) OutputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(&item{}),
		reflect.TypeOf(&model.Packet{}),
	}
}

func TestMuxerNode_Action(t *testing.T) {
	muxerNode := &MuxerNode{BaseNode: pipeline.NewBaseNode()}

	p1 := &printNode{BaseNode: pipeline.NewBaseNode(), idx: 1}
	p2 := &printNode{BaseNode: pipeline.NewBaseNode(), idx: 2}
	p3 := &printNode{BaseNode: pipeline.NewBaseNode(), idx: 3}

	muxerNode.PutSinkNode(p1)
	muxerNode.PutSinkNode(p2)
	muxerNode.PutSinkNode(p3)

	mockPipe.MountFirst(muxerNode, false)

	_, err := mockPipe.Check()
	if err != nil {
		t.Fatal(err)
	}
	p := &model.Packet{}
	p.MediaType = base.MediaDataTypeVideo
	mockPipe.Fire(context.Background(), p)
}
