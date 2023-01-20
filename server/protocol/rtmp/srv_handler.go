package rtmp

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/protocol/data_format/amf"
	"github.com/Opafanls/hylan/server/session"
	"github.com/Opafanls/hylan/server/stream"
	"io"
)

type RtmpHandler struct {
	*session.BaseHandler
	handshake   *handshake
	chunkState  *chunkState
	chunkStream map[int]*chunkStream
	poolBuf     pool.BufPool
	amfEncoder  *amf.Encoder
	amfDecoder  *amf.Decoder

	connDone bool
	publish  bool
}

func NewRtmpHandler(sess session.HySessionI) *RtmpHandler {
	rh := &RtmpHandler{BaseHandler: &session.BaseHandler{}}
	rh.handshake = newHandshake()
	rh.chunkState = newChunkState()
	rh.Conn = sess.GetConn()
	rh.Sess = sess
	rh.poolBuf = pool.P()
	rh.chunkState.streamID = 1
	rh.chunkStream = make(map[int]*chunkStream)
	rh.amfDecoder = amf.NewDecoder()
	rh.amfEncoder = amf.NewEncoder()
	rh.Base = base.NewEmptyBase()
	return rh
}

func (rh *RtmpHandler) OnStart(ctx context.Context, sess session.HySessionI) (base.StreamBaseI, error) {
	var err error
	rh.Ctx = ctx
	rh.Sess = sess
	err = rh.handshake.handshake(rh.Conn)
	if err != nil {
		return nil, err
	}
	err = rh.msgLoop(false)
	if err != nil {
		return nil, err
	}
	if rh.publish {
		sess.SetConfig(constdef.ConfigKeySessionType, constdef.SessionTypeRtmpSource)
	} else {
		sess.SetConfig(constdef.ConfigKeySessionType, constdef.SessionTypeRtmpSink)
	}
	return rh.Base, nil
}

func (rh *RtmpHandler) OnStop() error {

	return nil
}

func (rh *RtmpHandler) decodeBasicHeader(buf []byte) (format, int, error) {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(rh.Conn, buf[:1], 1)
	if err != nil {
		return invalid, 0, err
	}
	chunkFmt := format(buf[0] >> 6)
	csID := int(buf[0] & 0x3f)
	switch csID {
	case 0:
		//1 byte
		_, err = io.ReadAtLeast(rh.Conn, buf[1:2], 1)
		if err != nil {
			return invalid, 0, err
		}
		csID = int(buf[1]) + 64
		break
	case 1:
		//2 bytes
		_, err = io.ReadAtLeast(rh.Conn, buf[1:], 2)
		if err != nil {
			return invalid, 0, err
		}
		csID = int(buf[2])*256 + int(buf[1]) + 64
		break
	}
	return chunkFmt, csID, nil
}

func (rh *RtmpHandler) onConnect(cs *chunkStream) {
}

func (rh *RtmpHandler) OnStreaming(ctx context.Context, info base.StreamBaseI, sess session.HySessionI) error {
	//pub source stream to stream_center
	err := rh.BaseHandler.OnStreaming(ctx, info, sess)
	if err != nil {
		return err
	}
	if rh.publish {
		rh.connDone = true
		return rh.msgLoop(true)
	} else {
		source := stream.DefaultHyStreamManager.GetStreamByID(info.ID())
		if source == nil {
			return fmt.Errorf("source not found")
		}
		remoteAddr, _ := sess.GetConn().GetConfig(hynet.RemoteAddr)
		return source.Sink(&session.SinkArg{
			Ctx:    ctx,
			Sink:   session.NewBaseSink(info, fmt.Sprintf("rtmp_%s", remoteAddr)),
			Remote: sess,
			Local:  source.Source(),
		})
	}
}
