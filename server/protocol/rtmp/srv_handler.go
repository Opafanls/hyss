package rtmp

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol"
	"github.com/Opafanls/hylan/server/session"
	"io"
)

type RtmpHandler struct {
	*session.BaseHandler
	ctx         context.Context
	conn        hynet.IHyConn
	sess        session.HySessionI
	handshake   *handshake
	chunkState  *chunkState
	chunkHeader *chunkHeader
	chunkStream *chunkStream
	poolBuf     pool.BufPool

	published bool
}

func NewRtmpHandler(sess session.HySessionI) *RtmpHandler {
	rh := &RtmpHandler{BaseHandler: &session.BaseHandler{}}
	rh.handshake = newHandshake()
	rh.chunkState = newChunkState()
	rh.chunkHeader = newChunkHeader()
	rh.conn = sess.GetConn()
	rh.sess = sess
	rh.poolBuf = pool.P()
	rh.chunkState.streamID = 1
	return rh
}

func (rh *RtmpHandler) OnInit(ctx context.Context) error {
	return nil
}

func (rh *RtmpHandler) OnStart(ctx context.Context, sess session.HySessionI) (base.StreamBaseI, error) {
	var err error
	rh.ctx = ctx
	rh.sess = sess
	err = rh.handshake.handshake(rh.conn)
	if err != nil {
		return nil, err
	}
	cs := rh.getChunkStream(rh.ctx)
	log.Infof(rh.ctx, "handshake done, get chunkStream %+v", cs)
	err = cs.msgLoop(false)
	if err != nil {
		return nil, err
	}
	return rh.chunkStream.base, nil
}

func (rh *RtmpHandler) OnMedia(ctx context.Context, w *protocol.MediaWrapper) error {

	return nil
}

func (rh *RtmpHandler) OnStop() error {

	return nil
}

func (rh *RtmpHandler) decodeBasicHeader(buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(rh.conn, buf[:1], 1)
	if err != nil {
		return err
	}
	basicHeader := rh.chunkHeader.basicChunkHeader
	basicHeader.fmt = format((buf[0] >> 6) & 0b0000_0011)
	csID := int(buf[0] & 0b0011_1111)
	switch csID {
	case 0:
		//1 byte
		_, err = io.ReadAtLeast(rh.conn, buf[1:2], 1)
		if err != nil {
			return err
		}
		csID = int(buf[1]) + 64
		break
	case 1:
		//2 bytes
		_, err = io.ReadAtLeast(rh.conn, buf[1:], 2)
		if err != nil {
			return err
		}
		csID = int(buf[2])*256 + int(buf[1]) + 64
		break
	}
	basicHeader.csID = csID
	return nil
}

func (rh *RtmpHandler) getChunkStream(ctx context.Context) *chunkStream {
	csID := rh.chunkHeader.basicChunkHeader.csID
	//cs, ok := rh.chunkStream[csID]
	//if !ok {
	//	cs = newChunkStream(log.GetCtxWithLogID(ctx, fmt.Sprintf("cs_id:%d", csID)), rh.conn, rh)
	//	cs.chunkStreamID = csID
	//	rh.chunkStream[csID] = cs
	//}
	if rh.chunkStream == nil {
		cs := newChunkStream(log.GetCtxWithLogID(ctx, fmt.Sprintf("cs_id:%d", csID)), rh.conn, rh)
		rh.chunkStream = cs
		cs.chunkStreamID = csID
	}
	return rh.chunkStream
}

func (rh *RtmpHandler) onConnect(cs *chunkStream) {
}

func (rh *RtmpHandler) OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess session.HySessionI) error {
	err := rh.BaseHandler.OnStreamPublish(ctx, info, sess)
	if err != nil {
		return err
	}
	rh.published = true
	return rh.chunkStream.msgLoop(true)
}
