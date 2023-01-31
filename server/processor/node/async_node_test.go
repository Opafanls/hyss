package node

import (
	"context"
	"github.com/Opafanls/hylan/server/core/pipeline"
	"testing"
	"time"
)

func TestAsyncNode_Action(t *testing.T) {
	asyncNode := &AsyncNode{
		BaseNode: pipeline.NewBaseNode(),
		ch:       make(chan interface{}, 12),
		sl:       0,
		handleFun: func(data interface{}) {
			t.Logf("%v", data)
		},
		stop: make(chan struct{}),
	}
	asyncNode.Init()
	mockPipe.MountFirst(asyncNode, false)

	mockPipe.Fire(context.Background(), "1")

	time.Sleep(1 * time.Second)
	asyncNode.Destroy()
}
