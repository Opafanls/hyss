package rtmp

import (
	"context"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol/data/amf"
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
	chunkStream map[int]*chunkStream
	poolBuf     pool.BufPool
	amfEncoder  *amf.Encoder
	amfDecoder  *amf.Decoder

	connDone bool
	publish  bool
	base     base.StreamBaseI
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
	rh.chunkStream = make(map[int]*chunkStream)
	rh.amfDecoder = amf.NewDecoder()
	rh.amfEncoder = amf.NewEncoder()
	rh.base = base.NewEmptyBase()
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
	err = rh.msgLoop(false)
	if err != nil {
		return nil, err
	}
	return rh.base, nil
}

func (rh *RtmpHandler) OnMedia(ctx context.Context, w *model.Packet) error {
	source := rh.sess
	err := source.Push(ctx, w)
	if err != nil {
		return err.Err
	}
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
	basicHeader.fmt = format(buf[0] >> 6)
	csID := int(buf[0] & 0x3f)
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

func (rh *RtmpHandler) onConnect(cs *chunkStream) {
}

func (rh *RtmpHandler) OnStreamPublish(ctx context.Context, info base.StreamBaseI, sess session.HySessionI) error {
	err := rh.BaseHandler.OnStreamPublish(ctx, info, sess)
	if err != nil {
		return err
	}
	rh.connDone = true
	return rh.msgLoop(true)
}
