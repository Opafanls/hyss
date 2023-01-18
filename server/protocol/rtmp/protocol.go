package rtmp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/core/bitio"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol"
	"github.com/Opafanls/hylan/server/protocol/data/amf"
	"io"
	"net/url"
	"strings"
	"time"
)

const controlStreamID = 0
const Version = 3

//The maximum chunk size defaults to 128 bytes,
//but the client or the server can change this value, and updates its peer using this message
const defaultMaxChunkSize = 128
const setChunkSize = 1024

type TypeID byte
type format byte
type rtmpCmd string
type LimitType byte

const (
	format0_timestamp_msglen_msgtypeid_msgstreamid format = iota
	format1_timestamp_delta_and_msg_info
	format2_only_timestamp_delta
	format3_nothing
)

const (
	TypeIDSetChunkSize            TypeID = 1
	TypeIDAbortMessage            TypeID = 2
	TypeIDAck                     TypeID = 3
	TypeIDUserCtrl                TypeID = 4
	TypeIDWinAckSize              TypeID = 5
	TypeIDSetPeerBandwidth        TypeID = 6
	TypeIDAudioMessage            TypeID = 8
	TypeIDVideoMessage            TypeID = 9
	TypeIDDataMessageAMF3         TypeID = 15
	TypeIDSharedObjectMessageAMF3 TypeID = 16
	TypeIDCommandMessageAMF3      TypeID = 17
	TypeIDDataMessageAMF0         TypeID = 18
	TypeIDSharedObjectMessageAMF0 TypeID = 19
	TypeIDCommandMessageAMF0      TypeID = 20
	TypeIDAggregateMessage        TypeID = 22
)

const (
	cmdConnect       rtmpCmd = "connect"
	cmdFcpublish     rtmpCmd = "FCPublish"
	cmdReleaseStream rtmpCmd = "releaseStream"
	cmdCreateStream  rtmpCmd = "createStream"
	cmdPublish       rtmpCmd = "publish"
	cmdFCUnpublish   rtmpCmd = "FCUnpublish"
	cmdDeleteStream  rtmpCmd = "deleteStream"
	cmdPlay          rtmpCmd = "play"

	respResult     = "_result"
	respError      = "_error"
	onStatus       = "onStatus"
	publishStart   = "publish_start"
	playStart      = "play_start"
	connectSuccess = "connect_success"
	onBWDone       = "on_bwdone"
)

const (
	Hard LimitType = iota
	Soft
	Dynamic
)

type S0C0 struct {
	ver byte
}

func (s0 *S0C0) encode(conn io.Writer) error {
	_, err := conn.Write([]byte{s0.ver})
	return err
}

func (s0 *S0C0) decode(conn io.Reader) error {
	buf := make([]byte, 1)
	_, err := io.ReadAtLeast(conn, buf, 1)
	if err != nil {
		return err
	}
	s0.ver = buf[0]
	return nil
}

type S1C1 struct {
	time   uint32
	zero   uint32
	random []byte

	clientEcho []byte
}

type C2 struct {
	serverSendRandom []byte
}

func (s1 *S1C1) encode(conn io.Writer, cache []byte) error {
	buf := cache[:4]
	binary.BigEndian.PutUint32(buf, s1.time)
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(buf, s1.zero)
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	if _, err := conn.Write(s1.random); err != nil {
		return err
	}
	return nil
}

func (s1 *S1C1) decode(conn io.Reader, buf []byte) error {
	_, err := io.ReadAtLeast(conn, buf, 1536)
	if err != nil {
		return err
	}
	b1 := buf[:4]
	b2 := buf[4:8]
	s1.time = binary.BigEndian.Uint32(b1)
	s1.zero = binary.BigEndian.Uint32(b2)
	s1.random = buf[8:1536]
	s1.clientEcho = make([]byte, len(s1.random))
	copy(s1.clientEcho, s1.random[:])
	return nil
}

func (s2 *C2) decodeAndAuth(conn io.Reader, buf []byte) error {
	_, err := io.ReadAtLeast(conn, buf[:1536], 1536)
	if err != nil {
		return err
	}
	//if !bytes.Equal(s2.serverSendRandom[:], buf[8:1536]) {
	//	return fmt.Errorf("auth failed")
	//}
	return nil
}

//The version defined by this specification is 3

type handshake struct {
	s0c0      *S0C0
	s1c1      *S1C1
	c2        *C2
	cacheBuff []byte
}

func newHandshake() *handshake {
	h := &handshake{
		s0c0: &S0C0{},
		s1c1: &S1C1{
			clientEcho: make([]byte, 1528),
		},
		c2: &C2{},
	}

	return h
}

func (hs *handshake) handshake(conn io.ReadWriter) error {
	buf := hs.cacheBuff
	if buf == nil {
		buf = make([]byte, 2*setChunkSize)
		hs.cacheBuff = buf
	}
	if err := hs.s0c0.decode(conn); err != nil {
		return err
	}
	if err := hs.s1c1.decode(conn, buf); err != nil {
		return err
	}
	//发送s0 version
	if err := hs.s0c0.encode(conn); err != nil {
		return err
	}
	hs.s1c1.time = uint32(time.Now().UnixNano() / int64(time.Millisecond))
	hs.s1c1.zero = 0
	//set s1 random bytes
	if _, err := rand.Read(hs.s1c1.random); err != nil {
		return err
	}
	if err := hs.s1c1.encode(conn, buf); err != nil {
		return err
	}
	tmp := hs.s1c1.time
	hs.s1c1.time = hs.s1c1.zero
	hs.s1c1.zero = tmp
	hs.s1c1.random = hs.s1c1.clientEcho
	if err := hs.s1c1.encode(conn, buf); err != nil {
		return err
	}
	hs.c2.serverSendRandom = hs.s1c1.random
	if err := hs.c2.decodeAndAuth(conn, buf); err != nil {
		return err
	}
	return nil
}

type chunkState struct {
	chunkSize           uint32
	clientChunkSize     uint32
	windowAckSize       uint32
	clientWindowAckSize uint32
	received            uint32
	ackReceived         uint32

	transactionID int
	streamID      int
}

func newChunkState() *chunkState {
	chunkState := &chunkState{
		chunkSize:       128,
		clientChunkSize: 128,
	}
	return chunkState
}

type chunkStream struct {
	h             *RtmpHandler
	ctx           context.Context
	conn          io.ReadWriter
	amfEncoder    *amf.Encoder
	amfDecoder    *amf.Decoder
	chunkStreamID int
	remain        uint32
	all           bool
	bufPool       pool.BufPool
	csBuffer      *bytes.Buffer

	base base.StreamBaseI
}

func newChunkStream(ctx context.Context, conn io.ReadWriter, h *RtmpHandler) *chunkStream {
	cs := &chunkStream{}
	cs.conn = conn
	cs.amfDecoder = amf.NewDecoder()
	cs.amfEncoder = amf.NewEncoder()
	cs.ctx = ctx
	cs.h = h
	cs.bufPool = h.poolBuf
	cs.csBuffer = bytes.NewBuffer(make([]byte, setChunkSize))
	cs.base = base.NewEmptyBase()
	return cs
}

func (cs *chunkStream) msgLoop(publish bool) error {
	for cs.h.published == publish {
		readChunkSize := cs.h.chunkState.clientChunkSize
		cs.csBuffer.Reset()
		err := cs.readChunk(readChunkSize)
		if err != nil {
			return err
		}
		switch cs.h.chunkHeader.messageTypeID {
		case TypeIDSetChunkSize:
			msg := cs.csBuffer.Bytes()
			clientCs := binary.BigEndian.Uint32(msg[:4])
			cs.h.chunkState.clientChunkSize = clientCs

		case TypeIDAbortMessage:

		case TypeIDAck:

		case TypeIDUserCtrl:
			err := cs.handleUserControl()
			if err != nil {
				return fmt.Errorf("handleUserControl err: %+v", err)
			}

		case TypeIDWinAckSize:

		case TypeIDSetPeerBandwidth:

		case TypeIDCommandMessageAMF0:
			err := cs.handleAMF0()
			if err != nil {
				return err
			}
		case TypeIDAudioMessage:

		case TypeIDVideoMessage:
			err := cs.handleVideo()
			if err != nil {
				return err
			}
		default:
			log.Infof(cs.ctx, "invalid message type %d", cs.h.chunkHeader.messageTypeID)
		}
	}
	return nil
}

func (cs *chunkStream) handleUserControl() error {

	return nil
}

func (cs *chunkStream) readChunk(chunkSize uint32) error {
	for !cs.all {
		//log.Infof(cs.ctx, "start decode basic header")
		err := cs.h.decodeBasicHeader(nil)
		if err != nil {
			return err
		}
		//log.Infof(cs.ctx, "decode basic header1 %+v", cs.h.chunkHeader)
		err = cs.h.decodeMessageHeader(nil)
		if err != nil {
			return err
		}
		//log.Infof(cs.ctx, "decode basic header2 %+v", cs.h.chunkHeader)
		hd := cs.h.chunkHeader
		if hd.fmt == format0_timestamp_msglen_msgtypeid_msgstreamid ||
			hd.fmt == format1_timestamp_delta_and_msg_info {
			cs.remain = hd.messageLen
		}
		readLen := cs.remain
		if readLen > chunkSize {
			readLen = chunkSize
			cs.remain -= readLen
		} else {
			cs.remain = 0
		}
		buf, err := cs.bufPool.Make(int(readLen))
		if err != nil {
			return err
		}
		_, err = io.ReadAtLeast(cs.conn, buf, int(readLen))
		if err != nil {
			return err
		}
		if cs.remain == 0 {
			cs.all = true
		}
		cs.csBuffer.Write(buf)
	}
	cs.all = false
	return nil
}

func (cs *chunkStream) handleAMF0() error {
	data := cs.csBuffer.Bytes()
	cs.csBuffer.Reset()
	decoded, err := cs.amfDecoder.DecodeBatch(bytes.NewReader(data), amf.AMF0)
	log.Infof(cs.ctx, "decode data: %+v", decoded)
	if err != nil && err != io.EOF {
		return err
	}
	if len(decoded) == 0 {
		log.Warnf(cs.ctx, "decoded msg is empty")
		return nil
	}
	cmd, ok := decoded[0].(string)
	if !ok {
		return fmt.Errorf("decode cmd is not string, but %v", decoded[0])
	}
	switch rtmpCmd(cmd) {
	case cmdConnect:
		return cs.handleConnect(decoded[1:])
	case cmdReleaseStream:
		return cs.handleReleaseStream(decoded[1:])
	case cmdFcpublish:
		return cs.handleFCPublish(decoded[1:])
	case cmdCreateStream:
		return cs.handleCreateStream(decoded[1:])
	case cmdPublish:
		err := cs.handlePublish(decoded[1:])
		if err != nil {
			return err
		}
		cs.h.published = true
	}

	return nil
}

/**
+-----------+--------+-----------------------------+----------------+
| Property  |  Type  |        Description          | Example Value  |
+-----------+--------+-----------------------------+----------------+
|   app     | String | The Server application name |    testapp     |
|           |        | the client is connected to. |                |
+-----------+--------+-----------------------------+----------------+
| flashver  | String | Flash Player version. It is |    FMSc/1.0    |
|           |        | the same string as returned |                |
|           |        | by the ApplicationScript    |                |
|           |        | getversion () function.     |                |
+-----------+--------+-----------------------------+----------------+
|  swfUrl   | String | URL of the source SWF file  | file://C:/     |
|           |        | making the connection.      | FlvPlayer.swf  |
+-----------+--------+-----------------------------+----------------+
|  tcUrl    | String | URL of the Server.          | rtmp://local   |
|           |        | It has the following format.| host:1935/test |
|           |        | protocol://servername:port/ | app/instance1  |
|           |        | appName/appInstance         |                |
+-----------+--------+-----------------------------+----------------+
|  fpad     | Boolean| True if proxy is being used.| true or false  |
+-----------+--------+-----------------------------+----------------+
|audioCodecs| Number | Indicates what audio codecs | SUPPORT_SND    |
|           |        | the client supports.        | _MP3           |
+-----------+--------+-----------------------------+----------------+
|videoCodecs| Number | Indicates what video codecs | SUPPORT_VID    |
|           |        | are supported.              | _SORENSON      |
+-----------+--------+-----------------------------+----------------+
|videoFunct-| Number | Indicates what special video| SUPPORT_VID    |
|ion        |        | functions are supported.    | _CLIENT_SEEK   |
+-----------+--------+-----------------------------+----------------+
|  pageUrl  | String | URL of the web page from    | http://        |
|           |        | where the SWF file was      | somehost/      |
|           |        | loaded.                     | sample.html    |
+-----------+--------+-----------------------------+----------------+
| object    | Number | AMF encoding method.        |     AMF3       |
| Encoding  |        |                             |                |
+-----------+--------+-----------------------------+----------------+
*/
func (cs *chunkStream) handleConnect(decoded []interface{}) error {
	for _, de := range decoded {
		switch de.(type) {
		case float64:
			cs.h.chunkState.transactionID = int(de.(float64))
			break
		case amf.Object:
			obj := de.(amf.Object)
			tcUrl := obj["tcUrl"]
			if tcUrlStr, ok := tcUrl.(string); ok {
				parsed, err := url.Parse(tcUrlStr)
				if err != nil {
					return fmt.Errorf("parse rtmp url err: %+v", err)
				}
				schema := parsed.Scheme
				if schema != "rtmp" {
					return fmt.Errorf("invalid schema %s", schema)
				}
				cs.base.SetParam(base.ParamKeyVhost, parsed.Host)
				cs.h.onConnect(cs)
			} else {
				return fmt.Errorf("invalid rtmp parse type tcUrl")
			}
		}
	}
	err := cs.writeChunk(initControlMsg(TypeIDWinAckSize, 4, 2500000, nil))
	if err != nil {
		return err
	}
	err = cs.writeChunk(initControlMsg(TypeIDSetPeerBandwidth, 5, 2500000, Dynamic)) //2=dynamic
	if err != nil {
		return err
	}
	err = cs.writeChunk(initControlMsg(TypeIDSetChunkSize, 4, 1024, nil))
	if err != nil {
		return err
	}
	h := cs.h.chunkHeader
	resp := make(amf.Object)
	resp["fmsVer"] = "FMS/3,0,1,123"
	resp["capabilities"] = 31

	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetConnection.Connect.Success"
	event["description"] = "Connection succeeded."
	//event["objectEncoding"] = connServer.ConnInfo.ObjectEncoding
	return cs.sendMsg(h.csID, h.messageStreamID, amf.AMF0, "_result", cs.h.chunkState.transactionID, resp, event)
}

func (cs *chunkStream) handleReleaseStream(decoded []interface{}) error {
	log.Infof(cs.ctx, "handleReleaseStream %+v", decoded)
	return nil
}

func (cs *chunkStream) handleFCPublish(decoded []interface{}) error {
	return nil
}

func (cs *chunkStream) handleCreateStream(decoded []interface{}) error {
	for _, v := range decoded {
		switch v.(type) {
		case string:
			break
		case float64:
			cs.h.chunkState.transactionID = int(v.(float64))
			break
		case amf.Object:
			break
		}
	}
	h := cs.h.chunkHeader
	s := cs.h.chunkState
	return cs.sendMsg(h.csID, h.messageStreamID, amf.AMF0, "_result", s.transactionID, nil, s.streamID)
}

func (cs *chunkStream) handlePublish(decoded []interface{}) error {
	for k, v := range decoded {
		switch v.(type) {
		case string:
			if k == 2 {
				//name and query
				//stream?a=1&b=1
				nameAndQuery := v.(string)
				idx := strings.Index(nameAndQuery, "?")
				if idx < 0 {
					//no query
					cs.base.SetParam(base.ParamKeyName, nameAndQuery)
				} else {
					cs.base.SetParam(base.ParamKeyName, nameAndQuery[:idx])
					parsedQuery, err := url.ParseQuery(nameAndQuery[idx+1:])
					if err != nil {
						return fmt.Errorf("parse query failed: %+v %s", err, nameAndQuery)
					}
					for k, v := range parsedQuery {
						cs.base.SetParam(k, v[0])
					}
				}
			} else if k == 3 {
				cs.base.SetParam(base.ParamKeyApp, v.(string))
			}
		case float64:
			id := int(v.(float64))
			cs.h.chunkState.transactionID = id
		case amf.Object:
		}
	}
	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetStream.Publish.Start"
	event["description"] = "Start publishing."
	h := cs.h
	hd := h.chunkHeader
	return cs.sendMsg(hd.csID, hd.messageStreamID, amf.AMF0, "onStatus", 0, nil, event)
}

func (cs *chunkStream) handleVideo() error {
	data := cs.csBuffer.Bytes()
	k1 := data[0]
	var frameType protocol.FrameType
	switch k1 >> 4 & 0x0F {
	case 2:
		frameType = protocol.P
	case 1:
		frameType = protocol.I
	default:
		frameType = protocol.B
	}
	var codec protocol.VCodec
	switch k1 & 0x0F {
	case 7:
		codec = protocol.H264
	default:
		codec = protocol.Unknown2
	}
	return cs.h.OnMedia(cs.ctx, &protocol.MediaWrapper{
		Data:          data[1:],
		FrameType:     frameType,
		VCodec:        codec,
		MediaDataType: protocol.MediaDataTypeVideo,
	})
}

func (cs *chunkStream) sendMsg(csID int, streamID uint32, v amf.Version, args ...interface{}) error {
	for _, arg := range args {
		_, err := cs.amfEncoder.Encode(cs.csBuffer, arg, v)
		if err != nil {
			return fmt.Errorf("encode %v failed: %+v", arg, err)
		}
	}
	msg := cs.csBuffer.Bytes()
	cs.csBuffer.Reset()
	chunkP := &chunkPayload{
		chunkHeader: &chunkHeader{
			basicChunkHeader: &basicChunkHeader{
				fmt:  format0_timestamp_msglen_msgtypeid_msgstreamid,
				csID: csID,
			},
			messageHeader: &messageHeader{
				timestamp:       0,
				messageLen:      uint32(len(msg)),
				messageTypeID:   TypeIDCommandMessageAMF0,
				messageStreamID: streamID,
			},
		},
		chunkData: msg,
	}
	return cs.writeChunk(chunkP)
}

func (cs *chunkStream) writeChunk(chunkData *chunkPayload) error {
	chunkSize := cs.h.chunkState.chunkSize
	h := chunkData.chunkHeader
	if h.messageTypeID == TypeIDAudioMessage {
		h.messageStreamID = 4
	} else if h.messageTypeID == TypeIDVideoMessage ||
		h.messageTypeID == TypeIDDataMessageAMF0 ||
		h.messageTypeID == TypeIDDataMessageAMF3 {
		h.messageStreamID = 6
	} else if h.messageTypeID == TypeIDSetChunkSize {
		cs.h.chunkState.chunkSize = binary.BigEndian.Uint32(chunkData.chunkData[:4])
	}

	writtenLen := uint32(0)
	numChunks := h.messageLen / chunkSize
	for i := uint32(0); i <= numChunks; i++ {
		if writtenLen >= h.messageLen {
			break
		}
		if i == 0 {
			h.fmt = format0_timestamp_msglen_msgtypeid_msgstreamid
		} else {
			h.fmt = format3_nothing
		}
		if err := cs.writeHeader(h); err != nil {
			return err
		}
		inc := chunkSize
		start := i * chunkSize
		if uint32(len(chunkData.chunkData))-start <= inc {
			inc = uint32(len(chunkData.chunkData)) - start
		}
		writtenLen += inc
		end := start + inc
		buf := chunkData.chunkData[start:end]
		if _, err := cs.conn.Write(buf); err != nil {
			return err
		}
	}

	return nil
}

func (cs *chunkStream) writeHeader(wh *chunkHeader) error {
	//basic header
	var err error
	h := byte(wh.fmt) << 6
	switch {
	case wh.csID < 0:
		return fmt.Errorf("invalid chunk streamID %d", wh.csID)
	case wh.csID < 64:
		h |= byte(wh.csID)
		_, err := bitio.WriteBE(cs.conn, h)
		if err != nil {
			return err
		}
	case wh.csID-64 < 256:
		h |= 0
		_, err := bitio.WriteBE(cs.conn, byte(1))
		if err != nil {
			return err
		}
		_, err = bitio.WriteLE(cs.conn, byte(wh.csID-64))
		if err != nil {
			return err
		}
	case wh.csID-64 < 65536:
		h |= 1
		_, err := bitio.WriteBE(cs.conn, byte(1))
		if err != nil {
			return err
		}
		_, err = bitio.WriteLE(cs.conn, uint16(wh.csID-64))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid chunk streamID %d", wh.csID)
	}
	//message header
	var ts uint32
	if wh.fmt == 3 {
		goto END
	}
	if wh.fmt == 2 {
		ts = wh.timestampDelta
		_, err = bitio.WriteUintBE(cs.conn, ts, 3)
		if err != nil {
			return err
		}
		goto END
	}
	ts = wh.timestamp
	if wh.timestamp > 0xffffff {
		ts = 0xffffff
	}
	_, err = bitio.WriteUintBE(cs.conn, ts, 3)
	if err != nil {
		return err
	}

	if wh.messageLen > 0xffffff {
		return fmt.Errorf("length=%d", wh.messageLen)
	}
	_, _ = bitio.WriteUintBE(cs.conn, wh.messageLen, 3)
	_, _ = bitio.WriteBE(cs.conn, byte(wh.messageTypeID))
	if wh.fmt == 1 {
		goto END
	}
	_, _ = bitio.WriteLE(cs.conn, wh.messageStreamID)
END:
	//Extended Timestamp
	if ts >= 0xffffff {
		_, _ = bitio.WriteBE(cs.conn, wh.timestamp)
	}
	return err
}

/**
+--------------+----------------+--------------------+--------------+
| Basic Header | Message Header | Extended Timestamp |  Chunk Data  |
+--------------+----------------+--------------------+--------------+
|                                                    |
|<------------------- Chunk Header ----------------->|
Chunk Format
*/
type chunkHeader struct {
	*basicChunkHeader
	*messageHeader
}

type chunkPayload struct {
	*chunkHeader
	chunkData []byte
}

func newChunkHeader() *chunkHeader {
	return &chunkHeader{
		basicChunkHeader: &basicChunkHeader{},
		messageHeader:    &messageHeader{},
	}
}

//size: msg size
//val: msg val
func initControlMsg(typeID TypeID, size, value uint32, e interface{}) *chunkPayload {
	if size < 4 {
		panic("new control msg should be greater than 3")
	}
	h := &chunkHeader{
		basicChunkHeader: &basicChunkHeader{
			fmt:  format0_timestamp_msglen_msgtypeid_msgstreamid,
			csID: 2,
		},
		messageHeader: &messageHeader{
			messageLen:      size,
			messageTypeID:   typeID,
			messageStreamID: controlStreamID, //control stream
		},
	}
	cp := &chunkPayload{}
	cp.chunkHeader = h
	cp.chunkData = make([]byte, size)
	binary.BigEndian.PutUint32(cp.chunkData[:4], value)
	switch size {
	case 5:
		cp.chunkData[4] = byte(e.(LimitType))
		break
	}
	return cp
}

type basicChunkHeader struct { //(1 to 3 bytes)
	fmt  format
	csID int //[2,65599]
}

type messageHeader struct { //(0, 3, 7, or 11 bytes)
	timestamp       uint32 //3 bytes
	timestampDelta  uint32
	messageLen      uint32 //3 byte
	messageTypeID   TypeID //1 byte
	messageStreamID uint32 //4byte
}

func (rh *RtmpHandler) decodeMessageHeader(buf []byte) error {
	fmt0 := rh.chunkHeader.basicChunkHeader.fmt
	switch fmt0 {
	case format0_timestamp_msglen_msgtypeid_msgstreamid:
		return rh.decodeFmtType0(buf)
	case format1_timestamp_delta_and_msg_info:
		return rh.decodeFmtType1(buf)
	case format2_only_timestamp_delta:
		return rh.decodeFmtType2(buf)
	case format3_nothing:
		return nil
	default:
		return fmt.Errorf("invalid basic header fmt %d", fmt0)
	}
}

/*
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                 timestamp                     |message length |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|      message length (cont)    |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            message stream id (cont)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
Chunk Message Header - Type 0
*/
func (rh *RtmpHandler) decodeFmtType0(buf []byte) error {
	if len(buf) < 11 {
		buf = make([]byte, 11)
	}
	_, err := io.ReadAtLeast(rh.conn, buf[:11], 11)
	if err != nil {
		return err
	}
	mh := rh.chunkHeader.messageHeader
	buf0 := make([]byte, 4)
	copy(buf0[1:], buf[:3]) // 24bits BE
	mh.timestamp = binary.BigEndian.Uint32(buf0)
	copy(buf0[1:], buf[3:6]) // 24bits BE
	mh.messageLen = binary.BigEndian.Uint32(buf0)
	mh.messageTypeID = TypeID(buf[6])
	mh.messageStreamID = binary.BigEndian.Uint32(buf[7:])
	mh.timestampDelta = 0
	if mh.timestamp == 0xffffff {
		cache32bits := make([]byte, 4)
		_, err := io.ReadAtLeast(rh.conn, cache32bits, 4)
		if err != nil {
			return err
		}
		//extend timestamp
		mh.timestamp = binary.BigEndian.Uint32(cache32bits)
	}
	return nil
}

/**
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                timestamp delta                |message length |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     message length (cont)     |message type id|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
Chunk Message Header - Type 1
*/
func (rh *RtmpHandler) decodeFmtType1(buf []byte) error {
	if len(buf) < 7 {
		buf = make([]byte, 7)
	}
	_, err := io.ReadAtLeast(rh.conn, buf, 7)
	if err != nil {
		return err
	}
	mh := rh.chunkHeader.messageHeader
	//stream id no change
	tmp := make([]byte, 4)
	copy(tmp[1:], buf[:3])
	mh.timestampDelta = binary.BigEndian.Uint32(tmp)
	copy(tmp[1:], buf[3:6])
	mh.messageLen = binary.BigEndian.Uint32(tmp)
	mh.messageTypeID = TypeID(buf[6])
	mh.timestamp += mh.timestampDelta
	return nil
}

/**
0                   1                   2
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|               timestamp delta                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
Chunk Message Header - Type 2
*/
func (rh *RtmpHandler) decodeFmtType2(buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(rh.conn, buf, 3)
	if err != nil {
		return err
	}
	mh := rh.chunkHeader.messageHeader
	tmp := make([]byte, 4)
	copy(tmp[1:], buf[:3])
	mh.timestampDelta = binary.BigEndian.Uint32(tmp)
	mh.timestamp += mh.timestampDelta
	return nil
}
