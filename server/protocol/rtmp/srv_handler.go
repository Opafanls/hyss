package rtmp

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol"
	"io"
)

type Handler struct {
	ctx                context.Context
	conn               hynet.IHyConn
	rtmpMessageHandler *rtmpHandler
}

type rtmpHandler struct {
	conn        hynet.IHyConn
	handshake   *handshake
	chunkState  *chunkState
	chunkHeader *chunkHeader
	chunkStream map[int]*chunkStream
	poolBuf     pool.BufPool

	transactionID int
	streamID      int
}

func NewRtmpHandler(ctx context.Context, conn hynet.IHyConn) *Handler {
	rtmpHandler := &rtmpHandler{}
	rtmpHandler.handshake = newHandshake()
	rtmpHandler.chunkStream = make(map[int]*chunkStream)
	rtmpHandler.chunkState = newChunkState()
	rtmpHandler.chunkHeader = newChunkHeader()
	rtmpHandler.conn = conn
	rtmpHandler.poolBuf = pool.P()
	rtmpHandler.streamID = 1
	h := &Handler{ctx: ctx, conn: conn, rtmpMessageHandler: rtmpHandler}
	return h
}

func (h *Handler) OnInit(ctx context.Context) error {
	var err error
	h.ctx = ctx
	err = h.handshake()
	if err != nil {
		return err
	}
	cs := h.rtmpMessageHandler.getChunkStream(h.ctx)
	log.Infof(h.ctx, "handshake done, get chunkStream %+v", cs)
	err = cs.msgLoop()
	return err
}

func (h *Handler) OnMedia(ctx context.Context, mediaType protocol.MediaDataType, data interface{}) error {

	return nil
}

func (h *Handler) OnClose() error {
	return h.conn.Close()
}

func (h *Handler) handshake() error {
	return h.rtmpMessageHandler.handshake.handshake(h.conn)
}

func (rh *rtmpHandler) decodeBasicHeader(buf []byte) error {
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

func (rh *rtmpHandler) getChunkStream(ctx context.Context) *chunkStream {
	csID := rh.chunkHeader.basicChunkHeader.csID
	cs, ok := rh.chunkStream[csID]
	if !ok {
		cs = newChunkStream(log.GetCtxWithLogID(ctx, fmt.Sprintf("cs_id:%d", csID)), rh.conn, rh)
		rh.chunkStream[csID] = cs
	}
	return cs
}
