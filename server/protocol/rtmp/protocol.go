package rtmp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/constdef"
	"github.com/Opafanls/hylan/server/core/bitio"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/model"
	"github.com/Opafanls/hylan/server/protocol/data_format/amf"
	"github.com/Opafanls/hylan/server/session"
	"github.com/Opafanls/hylan/server/stream"
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
	invalid
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
	cmdConnect         rtmpCmd = "connect"
	cmdFcpublish       rtmpCmd = "FCPublish"
	cmdReleaseStream   rtmpCmd = "releaseStream"
	cmdCreateStream    rtmpCmd = "createStream"
	cmdPublish         rtmpCmd = "publish"
	cmdFCUnpublish     rtmpCmd = "FCUnpublish"
	cmdDeleteStream    rtmpCmd = "deleteStream"
	cmdPlay            rtmpCmd = "play"
	cmdGetStreamLength rtmpCmd = "getStreamLength"

	respResult     = "_result"
	respError      = "_error"
	onStatus       = "onStatus"
	publishStart   = "publish_start"
	playStart      = "play_start"
	connectSuccess = "connect_success"
	onBWDone       = "on_bwdone"
)

const (
	streamBegin      uint32 = 0
	streamEOF        uint32 = 1
	streamDry        uint32 = 2
	setBufferLen     uint32 = 3
	streamIsRecorded uint32 = 4
	pingRequest      uint32 = 6
	pingResponse     uint32 = 7
)

const (
	Hard LimitType = iota
	Soft
	Dynamic
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
		sess.SetConfig(constdef.ConfigKeySinkRW, NewRtmp(rh))
		return source.Sink(&session.SinkArg{
			Ctx:    ctx,
			Sink:   session.NewBaseSink(info, fmt.Sprintf("rtmp_%s", remoteAddr)),
			Remote: sess,
			Local:  source.Source(),
		})
	}
}

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
	chunkStreamID int
	remain        uint32
	bufPool       pool.BufPool
	readBuffer    *bytes.Buffer
	writeBuffer   *bytes.Buffer
	v_recv        uint64
	a_recv        uint64

	ext bool
	*chunkHeader
}

func (cs *chunkStream) preceding() {
	cs.remain = cs.chunkHeader.msgLen
}

func newChunkStream(ctx context.Context, r *RtmpHandler, fmt format, csID int) *chunkStream {
	cs := &chunkStream{}
	cs.h = r
	cs.ctx = ctx
	cs.chunkHeader = newChunkHeader()
	cs.chunkHeader.fmt = fmt
	cs.chunkHeader.csID = csID
	cs.writeBuffer = bytes.NewBuffer(make([]byte, 0, setChunkSize))
	cs.readBuffer = bytes.NewBuffer(make([]byte, 0, setChunkSize))
	return cs
}

func (rh *RtmpHandler) msgLoop(init bool) error {
	for rh.connDone == init {
		readChunkSize := rh.chunkState.clientChunkSize
		cs, err := rh.readChunk(readChunkSize)
		if err != nil {
			return err
		}
		if err := rh.processChunk(cs); err != nil {
			return err
		}
		cs.readBuffer.Reset()
		cs.writeBuffer.Reset()
	}
	return nil
}

func (rh *RtmpHandler) processChunk(cs *chunkStream) error {
	switch cs.chunkHeader.typeID {
	case TypeIDSetChunkSize:
		msg := cs.readBuffer.Bytes()
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
	case TypeIDDataMessageAMF0:
	default:
		log.Infof(cs.ctx, "invalid message type %d", cs.chunkHeader.typeID)
	}
	return nil
}

func (cs *chunkStream) handleUserControl() error {
	data := cs.readBuffer.Bytes()
	if len(data) < 2 {
		return fmt.Errorf("invalid user control event")
	}

	event := uint32(data[0])<<8 + uint32(data[1])
	log.Infof(cs.ctx, "recv user control msg %d", event)
	switch event {
	case setBufferLen:
	}
	return nil
}

func (rh *RtmpHandler) readChunk(chunkSize uint32) (*chunkStream, error) {
	var done = false
	var csRet *chunkStream
	for !done {
		//decode format and chunk streamID
		format, csID, err := rh.decodeBasicHeader(nil)
		if err != nil {
			return nil, err
		}
		var chunkStream *chunkStream
		if cs0, ok := rh.chunkStream[csID]; !ok {
			cs0 = newChunkStream(rh.Ctx, rh, format, csID)
			rh.chunkStream[csID] = cs0
			chunkStream = cs0
		} else {
			cs0.chunkHeader.basicChunkHeader.fmt = format
			chunkStream = cs0
		}
		err = rh.decodeMessageHeader(chunkStream, nil)
		if err != nil {
			return nil, err
		}
		if format == format0_timestamp_msglen_msgtypeid_msgstreamid ||
			format == format1_timestamp_delta_and_msg_info {
			chunkStream.remain = chunkStream.chunkHeader.msgLen
		} else {
			if chunkStream.remain == 0 {
				chunkStream.preceding()
			}
		}
		readLen := chunkStream.remain
		if readLen > chunkSize {
			readLen = chunkSize
			chunkStream.remain -= readLen
		} else {
			chunkStream.remain = 0
		}
		buf, err := rh.poolBuf.Make(int(readLen))
		if err != nil {
			return nil, err
		}
		_, err = io.ReadAtLeast(rh.Conn, buf, int(readLen))
		if err != nil {
			return nil, err
		}
		chunkStream.readBuffer.Write(buf[:readLen])
		if chunkStream.remain == 0 {
			done = true
			csRet = chunkStream
			break
		}
	}
	return csRet, nil
}

func (cs *chunkStream) handleAMF0() error {
	data := cs.readBuffer.Bytes()
	cs.readBuffer.Reset()
	decoded, err := cs.h.amfDecoder.DecodeBatch(bytes.NewReader(data), amf.AMF0)
	log.Infof(cs.ctx, "decode data_format: %+v", decoded)
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
		cs.h.connDone = true
		cs.h.publish = true
	case cmdGetStreamLength:
	case cmdPlay:
		err := cs.handlePlay(decoded[1:])
		if err != nil {
			return err
		}
		cs.h.connDone = true
		cs.h.publish = false
	case cmdDeleteStream:

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
			/**
			+-----------+--------+-----------------------------+----------------+
			|  tcUrl    | String | URL of the Server.          | rtmp://local   |
			|           |        | It has the following format.| host:1935/test |
			|           |        | protocol://servername:port/ | app/instance1  |
			|           |        | appName/appInstance         |                |
			+-----------+--------+-----------------------------+----------------+
			*/
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
				cs.h.Base.SetParam(base.ParamKeyVhost, parsed.Host)
				cs.h.onConnect(cs)
			} else {
				return fmt.Errorf("invalid rtmp parse type tcUrl")
			}
			app := obj["app"]
			if appStr, ok := app.(string); ok {
				cs.h.Base.SetParam(base.ParamKeyApp, appStr)
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
	h := cs.chunkHeader
	resp := make(amf.Object)
	resp["fmsVer"] = "FMS/3,0,1,123"
	resp["capabilities"] = 31

	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetConnection.Connect.Success"
	event["description"] = "Connection succeeded."
	//event["objectEncoding"] = connServer.ConnInfo.ObjectEncoding
	return cs.sendMsg(h.csID, h.streamID, amf.AMF0, "_result", cs.h.chunkState.transactionID, resp, event)
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
	h := cs.chunkHeader
	s := cs.h.chunkState
	return cs.sendMsg(h.csID, h.streamID, amf.AMF0, "_result", s.transactionID, nil, s.streamID)
}

func (cs *chunkStream) handlePublish(decoded []interface{}) error {
	cs.h.Base.SetParam(base.ParamKeyIsSource, "1")
	for k, v := range decoded {
		switch v.(type) {
		case string:
			if k == 2 {
				//name and query  //stream?a=1&b=1
				nameAndQuery := v.(string)
				idx := strings.Index(nameAndQuery, "?")
				if idx < 0 {
					//without query
					cs.h.Base.SetParam(base.ParamKeyName, nameAndQuery)
				} else {
					cs.h.Base.SetParam(base.ParamKeyName, nameAndQuery[:idx])
					parsedQuery, err := url.ParseQuery(nameAndQuery[idx+1:])
					if err != nil {
						return fmt.Errorf("parse query failed: %+v %s", err, nameAndQuery)
					}
					for k, v := range parsedQuery {
						cs.h.Base.SetParam(k, v[0])
					}
				}
			} else if k == 3 {
				cs.h.Base.SetParam(base.ParamKeyApp, v.(string))
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
	hd := cs.chunkHeader
	return cs.sendMsg(hd.csID, hd.streamID, amf.AMF0, "onStatus", 0, nil, event)
}

func (cs *chunkStream) handlePlay(decoded []interface{}) error {
	for k, v := range decoded {
		switch v.(type) {
		case string:
			if k == 2 {
				cs.h.Base.SetParam(base.ParamKeyName, v.(string))
			} else if k == 3 {

			}
		case float64:
			id := int(v.(float64))
			cs.h.chunkState.transactionID = id
		case amf.Object:
		}
	}
	if err := cs.streamRecord(); err != nil {
		return fmt.Errorf("stream record err: %+v", err)
	}
	if err := cs.streamBegin(); err != nil {
		return fmt.Errorf("stream begin err: %+v", err)
	}

	return nil
}

func (cs *chunkStream) handleVideo() error {
	cs.v_recv++
	cs.h.Sess.Stat().IncrVideoPkt(1)
	data := cs.readBuffer.Bytes()
	if len(data) == 0 {
		return nil
	}
	k1 := data[0]
	var frameType constdef.FrameType
	switch k1 >> 4 & 0x0F {
	case 2:
		frameType = constdef.P
	case 1:
		frameType = constdef.I
	default:
		frameType = constdef.B
	}
	var codec constdef.VCodec
	switch k1 & 0x0F {
	case 7:
		codec = constdef.H264
	default:
		codec = constdef.Unknown2
	}
	return cs.h.OnMedia(cs.ctx, &model.Packet{
		FrameIdx:  cs.v_recv,
		MediaType: constdef.MediaDataTypeVideo,
		Timestamp: uint64(cs.timestamp),
		Data:      data[1:],
		Header:    NewRtmpPacketMeta(model.NewVideoPktH(frameType, codec, "rtmp"), cs.streamID),
	})
}

func (cs *chunkStream) sendMsg(csID int, streamID uint32, v amf.Version, args ...interface{}) error {
	defer cs.writeBuffer.Reset()
	wb := cs.writeBuffer
	for _, arg := range args {
		_, err := cs.h.amfEncoder.Encode(wb, arg, v)
		if err != nil {
			return fmt.Errorf("encode %v failed: %+v", arg, err)
		}
	}
	msg := wb.Bytes()
	chunkP := &chunkPayload{
		chunkHeader: &chunkHeader{
			basicChunkHeader: &basicChunkHeader{
				fmt:  format0_timestamp_msglen_msgtypeid_msgstreamid,
				csID: csID,
			},
			messageHeader: &messageHeader{
				timestamp: 0,
				msgLen:    uint32(len(msg)),
				typeID:    TypeIDCommandMessageAMF0,
				streamID:  streamID,
			},
		},
		chunkData: msg,
	}
	return cs.writeChunk(chunkP)
}

func (cs *chunkStream) writeChunk(chunkData *chunkPayload) error {
	chunkSize := cs.h.chunkState.chunkSize
	h := chunkData.chunkHeader
	if h.typeID == TypeIDAudioMessage {
		h.streamID = 4
	} else if h.typeID == TypeIDVideoMessage ||
		h.typeID == TypeIDDataMessageAMF0 ||
		h.typeID == TypeIDDataMessageAMF3 {
		h.streamID = 6
	} else if h.typeID == TypeIDSetChunkSize {
		cs.h.chunkState.chunkSize = binary.BigEndian.Uint32(chunkData.chunkData[:4])
	}

	writtenLen := uint32(0)
	numChunks := h.msgLen / chunkSize
	for i := uint32(0); i <= numChunks; i++ {
		if writtenLen >= h.msgLen {
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
		if _, err := cs.h.Conn.Write(buf); err != nil {
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
		_, err := bitio.WriteBE(cs.h.Conn, h)
		if err != nil {
			return err
		}
	case wh.csID-64 < 256:
		h |= 0
		_, err := bitio.WriteBE(cs.h.Conn, byte(1))
		if err != nil {
			return err
		}
		_, err = bitio.WriteLE(cs.h.Conn, byte(wh.csID-64))
		if err != nil {
			return err
		}
	case wh.csID-64 < 65536:
		h |= 1
		_, err := bitio.WriteBE(cs.h.Conn, byte(1))
		if err != nil {
			return err
		}
		_, err = bitio.WriteLE(cs.h.Conn, uint16(wh.csID-64))
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
		_, err = bitio.WriteUintBE(cs.h.Conn, ts, 3)
		if err != nil {
			return err
		}
		goto END
	}
	ts = wh.timestamp
	if wh.timestamp > 0xffffff {
		ts = 0xffffff
	}
	_, err = bitio.WriteUintBE(cs.h.Conn, ts, 3)
	if err != nil {
		return err
	}

	if wh.msgLen > 0xffffff {
		return fmt.Errorf("length=%d", wh.msgLen)
	}
	_, _ = bitio.WriteUintBE(cs.h.Conn, wh.msgLen, 3)
	_, _ = bitio.WriteBE(cs.h.Conn, byte(wh.typeID))
	if wh.fmt == 1 {
		goto END
	}
	_, _ = bitio.WriteLE(cs.h.Conn, wh.streamID)
END:
	//Extended Timestamp
	if ts >= 0xffffff {
		_, _ = bitio.WriteBE(cs.h.Conn, wh.timestamp)
	}
	return err
}

/*
   +------------------------------+-------------------------
   |     Event Type ( 2- bytes )  | Event Data
   +------------------------------+-------------------------
   Pay load for the ‘User Control Message’.
*/
func (cs *chunkStream) initUserControl(eventType, bodyLen uint32) *chunkPayload {
	var ret = newEmptyChunkPayload()
	bodyLen += 2
	ret.fmt = 0
	ret.csID = 2
	ret.typeID = TypeIDUserCtrl
	ret.streamID = 1
	ret.msgLen = bodyLen
	ret.chunkData = make([]byte, bodyLen)
	ret.chunkData[0] = byte(eventType >> 8 & 0xff)
	ret.chunkData[1] = byte(eventType & 0xff)
	return ret
}

func (cs *chunkStream) streamBegin() error {
	ret := cs.initUserControl(streamBegin, 4)
	for i := 0; i < 4; i++ {
		ret.chunkData[2+i] = byte(1 >> uint32((3-i)<<3) & 0xff)
	}
	return cs.writeChunk(ret)
}

func (cs *chunkStream) streamRecord() error {
	ret := cs.initUserControl(streamIsRecorded, 4)
	for i := 0; i < 4; i++ {
		ret.chunkData[2+i] = byte(1 >> uint32((3-i)<<3) & 0xff)
	}
	return cs.writeChunk(ret)
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

func newEmptyChunkPayload() *chunkPayload {
	return &chunkPayload{
		chunkHeader: newChunkHeader(),
		chunkData:   nil,
	}
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
			msgLen:   size,
			typeID:   typeID,
			streamID: controlStreamID, //control stream
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
	timestamp      uint32 //3 bytes
	timestampDelta uint32
	msgLen         uint32 //3 byte
	typeID         TypeID //1 byte
	streamID       uint32 //4byte
}

func (rh *RtmpHandler) decodeMessageHeader(chunkStream *chunkStream, buf []byte) error {
	fmt0 := chunkStream.fmt
	switch fmt0 {
	case format0_timestamp_msglen_msgtypeid_msgstreamid:
		return rh.decodeFmtType0(chunkStream, buf)
	case format1_timestamp_delta_and_msg_info:
		return rh.decodeFmtType1(chunkStream, buf)
	case format2_only_timestamp_delta:
		/**Type 2 chunk headers are 3 bytes long.
		Neither the stream ID nor the message length is included;
		this chunk has the same stream ID and message length as the preceding chunk.
		Streams with constant-sized messages (for example, some audio and data formats) SHOULD use this format for the first chunk of each message after the first.
		*/
		return rh.decodeFmtType2(chunkStream, buf)
	case format3_nothing:
		/**Type 3 chunks have no message header. The stream ID, message length and timestamp delta fields are not present;
		chunks of this type take values from the preceding chunk for the same Chunk Stream ID.
		When a single message is split into chunks, all chunks of a message except the first one SHOULD use this type.
		*/
		return rh.decodeFmtType3(chunkStream)
		//return nil
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
func (rh *RtmpHandler) decodeFmtType0(cs *chunkStream, buf []byte) error {
	if len(buf) < 11 {
		buf = make([]byte, 11)
	}
	_, err := io.ReadAtLeast(rh.Conn, buf[:11], 11)
	if err != nil {
		return err
	}
	mh := cs.messageHeader
	buf0 := make([]byte, 4)
	copy(buf0[1:], buf[:3]) // 24bits BE
	mh.timestamp = binary.BigEndian.Uint32(buf0)
	copy(buf0[1:], buf[3:6]) // 24bits BE
	mh.msgLen = binary.BigEndian.Uint32(buf0)
	mh.typeID = TypeID(buf[6])
	mh.streamID = binary.BigEndian.Uint32(buf[7:])
	mh.timestampDelta = 0
	if mh.timestamp == 0xffffff {
		cache32bits := make([]byte, 4)
		_, err := io.ReadAtLeast(rh.Conn, cache32bits, 4)
		if err != nil {
			return err
		}
		//extend timestamp
		mh.timestampDelta = binary.BigEndian.Uint32(cache32bits)
		mh.timestamp += mh.timestampDelta
		cs.ext = true
	} else {
		cs.ext = false
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
func (rh *RtmpHandler) decodeFmtType1(cs *chunkStream, buf []byte) error {
	if len(buf) < 7 {
		buf = make([]byte, 7)
	}
	_, err := io.ReadAtLeast(rh.Conn, buf, 7)
	if err != nil {
		return err
	}
	mh := cs.messageHeader
	//stream id no change
	tmp := make([]byte, 4)
	copy(tmp[1:], buf[:3])
	mh.timestampDelta = binary.BigEndian.Uint32(tmp)
	copy(tmp[1:], buf[3:6])
	mh.msgLen = binary.BigEndian.Uint32(tmp)
	mh.typeID = TypeID(buf[6])
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
func (rh *RtmpHandler) decodeFmtType2(cs *chunkStream, buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(rh.Conn, buf, 3)
	if err != nil {
		return err
	}
	mh := cs.messageHeader
	tmp := make([]byte, 4)
	copy(tmp[1:], buf[:3])
	mh.timestampDelta = binary.BigEndian.Uint32(tmp)
	mh.timestamp += mh.timestampDelta
	return nil
}

func (rh *RtmpHandler) decodeFmtType3(chunkStream *chunkStream) error {
	if chunkStream.remain == 0 {
		switch chunkStream.fmt {
		case format0_timestamp_msglen_msgtypeid_msgstreamid:
			if chunkStream.ext {
				buf := make([]byte, 4)
				_, err := io.ReadAtLeast(rh.Conn, buf, 4)
				if err != nil {
					return err
				}
				chunkStream.timestamp = binary.BigEndian.Uint32(buf)
			}
		case format1_timestamp_delta_and_msg_info, format2_only_timestamp_delta:
			var timedelta uint32
			if chunkStream.ext {
				buf := make([]byte, 4)
				_, err := io.ReadAtLeast(rh.Conn, buf, 4)
				if err != nil {
					return err
				}
				timedelta = binary.BigEndian.Uint32(buf)
			} else {
				timedelta = chunkStream.timestampDelta
			}
			chunkStream.timestamp += timedelta
		}
	} else {
		if chunkStream.ext {
			//b, err := rh.Conn.(4)
			//if err != nil {
			//	return err
			//}
			//tmpts := binary.BigEndian.Uint32(b)
			//if tmpts == chunkStream.Timestamp {
			//	r.Discard(4)
			//}
		}
	}
	return nil
}
