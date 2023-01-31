package pipeline

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type inputStru struct {
	name string
	val  int
}

type node1 struct {
	*BaseNode
}

func (n *node1) Init()          {}
func (n *node1) Destroy() error { return nil }
func (n *node1) Name() string {
	return "node1"
}

func (n *node1) Action(ctx context.Context, input interface{}) (output interface{}, err error) {
	switch in := input.(type) {
	case string:
		fmt.Printf("action input string is %s\n", in)
	case int32:
		fmt.Printf("action input int32 is %d\n", in)
	case *inputStru:
		fmt.Printf("action input inputStru is :%+v\n", in)
	}
	return "hello sink node\n", nil
}

func (n *node1) InputType() []reflect.Type {
	var t1 int32 = 1
	return []reflect.Type{
		reflect.TypeOf(t1),
		reflect.TypeOf(""),
		reflect.TypeOf(&inputStru{}),
	}
}

func (n *node1) OutputType() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(""),
		//reflect.TypeOf(1.0000),
	}
}

func (n *node1) PutSinkNode(sinkNode INode) error {
	return n.BaseNode.PutSinkNode(sinkNode)
}

func (n *node1) ListSinkNodes() []INode {
	return n.BaseNode.ListSinkNodes()
}

func TestDefaultPipeline_Check(t *testing.T) {
	d := NewDefaultP()
	n1 := &node1{NewBaseNode()}
	n2 := &node1{NewBaseNode()}

	if err := n1.PutSinkNode(n2); err != nil {
		t.Fatal(err)
	}

	if err := d.MountFirst(n1, false); err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", d.Print())

	_, err := d.Check()
	require.Nil(t, err)

	err = d.Fire(context.Background(), "1")
	require.Nil(t, err)

	t.Logf("%s", "")
}
