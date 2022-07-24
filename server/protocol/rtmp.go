package protocol

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/proto"
	"github.com/Opafanls/hylan/server/session"
	"github.com/Opafanls/hylan/server/transport"
	"github.com/notedit/rtmp/av"
	"github.com/notedit/rtmp/format/rtmp"
)

type RtmpServerProtocol struct {
	*BaseProtocol
	tcpTransport transport.Transport
	rtmpSess     *rtmp.Conn

	readBuffSize  int
	writeBuffSize int
}

func (r *RtmpServerProtocol) Name() string {
	return "rtmp_server"
}

func (r *RtmpServerProtocol) Listen(arg *ListenArg) {
	listener := &transport.ConnListener{}
	go func() {
		if err := listener.TcpListen(arg.Ip, arg.Port, r.handleTcpConnect); err != nil {
			panic(fmt.Errorf("rtmp server listen err %+v", err))
		}
	}()
}

func (r *RtmpServerProtocol) handleTcpConnect(ctx context.Context, transport transport.Transport) {
	rtmpConn := rtmp.NewConn(&bufio.ReadWriter{
		Reader: bufio.NewReaderSize(transport.GetConn(), r.readBuffSize),
		Writer: bufio.NewWriterSize(transport.GetConn(), r.writeBuffSize),
	})
	base := &BaseProtocol{ctx: ctx}
	rtmpSession := &RtmpServerProtocol{base, transport, rtmpConn, 4096, 4096}
	sess := session.NewHySession(ctx, rtmpSession)
	sess.Cycle()
}

func (r *RtmpServerProtocol) SetData(packet proto.PacketI) *constdef.HyError {
	err := r.rtmpSess.WritePacket(av.Packet{})
	if err != nil {
		return constdef.NewHyError("write_packet_failed", err)
	}
	return nil
}

func (r *RtmpServerProtocol) GetData() (proto.PacketI, *constdef.HyError) {
	pkt, err := r.rtmpSess.ReadPacket()
	if err != nil {
		return nil, constdef.NewHyError("read_packet_failed", err)
	}
	retPkt := &proto.RtmpPacket{Pkt: pkt}
	return retPkt, nil
}

func (r *RtmpServerProtocol) Stop(msg string) *constdef.HyError {
	err := r.tcpTransport.Close(r.ctx, msg)
	if err != nil {
		log.Errorf(r.ctx, "stop transport failed: %+v", err)
	}

	return nil
}
