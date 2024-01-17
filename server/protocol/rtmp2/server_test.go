package rtmp2

import (
	"github.com/torresjeff/rtmp"
	"go.uber.org/zap"
	"testing"
)

func TestRtmpServer(t *testing.T) {
	server := &rtmp.Server{}
	server.Logger = zap.L()
	server.Addr = "127.0.0.1:1935"
	t.Fatal(server.Listen())
}
