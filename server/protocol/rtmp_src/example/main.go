package main

import (
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp"
	"net"
)

func main() {
	var rtmpServer *rtmp.Server
	stream := rtmp.NewRtmpStream()
	rtmpServer = rtmp.NewRtmpServer(stream, nil)
	rtmpListen, err := net.Listen("tcp", ":1935")
	if err != nil {
		panic(err)
	}
	err = rtmpServer.Serve(rtmpListen)
	if err != nil {
		panic(err)
		return
	}
}
