package rtmp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/Opafanls/hylan/server/core/pool"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/protocol/amf"
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

type TypeID byte
type state string
type format byte
type rtmpCmd string

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

var (
	cmdConnect       rtmpCmd = "connect"
	cmdFcpublish     rtmpCmd = "FCPublish"
	cmdReleaseStream rtmpCmd = "releaseStream"
	cmdCreateStream  rtmpCmd = "createStream"
	cmdPublish       rtmpCmd = "publish"
	cmdFCUnpublish   rtmpCmd = "FCUnpublish"
	cmdDeleteStream  rtmpCmd = "deleteStream"
	cmdPlay          rtmpCmd = "play"
)

var (
	respResult     = "_result"
	respError      = "_error"
	onStatus       = "on_status"
	publishStart   = "publish_start"
	playStart      = "play_start"
	connectSuccess = "connect_success"
	onBWDone       = "on_bwdone"
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
	_, err := io.ReadAtLeast(conn, buf, 1536)
	if err != nil {
		return err
	}
	b0 := buf[1534]
	b1 := buf[1535]
	//if !bytes.Equal(s2.serverSendRandom[:], buf[8:1536]) {
	//	return fmt.Errorf("auth failed")
	//}
	log.Infof(context.Background(), "%+v %+v", b0, b1)
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
		buf = make([]byte, 2*1024)
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
}

func newChunkState() *chunkState {
	chunkState := &chunkState{
		chunkSize:       128,
		clientChunkSize: 128,
	}
	return chunkState
}

type chunkStream struct {
	h          *rtmpHandler
	ctx        context.Context
	conn       io.ReadWriter
	amfEncoder *amf.Encoder
	amfDecoder *amf.Decoder

	remain    uint32
	all       bool
	bufPool   pool.BufPool
	chunkData bytes.Buffer

	metadata map[string]interface{}

	query url.Values
}

func newChunkStream(ctx context.Context, conn io.ReadWriter, h *rtmpHandler) *chunkStream {
	cs := &chunkStream{}
	cs.conn = conn
	cs.amfDecoder = amf.NewDecoder()
	cs.amfEncoder = amf.NewEncoder()
	cs.ctx = ctx
	cs.h = h
	cs.bufPool = h.poolBuf
	return cs
}

func (cs *chunkStream) msgLoop() error {
	chunkSize := cs.h.chunkState.clientChunkSize
	for {
		err := cs.readChunk(chunkSize)
		if err != nil {
			return err
		}
		switch cs.h.chunkHeader.messageTypeID {
		case TypeIDSetChunkSize:
			break
		case TypeIDAbortMessage:
			break
		case TypeIDAck:
			break
		case TypeIDUserCtrl:
			break
		case TypeIDWinAckSize:
			break
		case TypeIDSetPeerBandwidth:
			break
		case TypeIDCommandMessageAMF0:
			return cs.handleAMF0()
		}
	}
}

func (cs *chunkStream) readChunk(chunkSize uint32) error {
	for !cs.all {
		err := cs.h.decodeBasicHeader(nil)
		if err != nil {
			return err
		}
		err = cs.h.decodeMessageHeader(nil)
		if err != nil {
			return err
		}
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
		cs.chunkData.Write(buf)
	}
	cs.all = false
	return nil
}

func (cs *chunkStream) handleAMF0() error {
	data := cs.chunkData.Bytes()
	decoded, err := cs.amfDecoder.DecodeBatch(bytes.NewReader(data), amf.AMF0)
	log.Infof(cs.ctx, "decode data: %+v", decoded)
	if err != nil && err != io.EOF {
		return err
	}
	if len(decoded) == 0 {
		log.Warnf(cs.ctx, "decoded msg is empty")
		return nil
	}
	cmd, ok := decoded[0].(rtmpCmd)
	if !ok {
		return fmt.Errorf("decode cmd is not string, but %v", decoded[0])
	}
	switch cmd {
	case cmdConnect:
		cs.handleConnect(decoded[1:])
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
			break
		case amf.Object:
			obj := de.(amf.Object)
			cs.metadata = obj
			app := obj["app"]
			if appStr, ok := app.(string); ok {
				tmp := strings.Split(appStr, "/")
				if len(tmp) != 2 {
					return fmt.Errorf("invalid rtmp app %s", appStr)
				}
			} else {
				return fmt.Errorf("invalid rtmp parse type app")
			}
			tcUrl := obj["tcUrl"]
			if tcUrlStr, ok := tcUrl.(string); ok {
				parsed, err := url.Parse(tcUrlStr)
				if err != nil {
					return fmt.Errorf("parse rtmp url err: %+v", err)
				}
				schema := parsed.Scheme
				query := parsed.Query()
				if schema != "rtmp" {
					return fmt.Errorf("invalid schema %s", schema)
				}
				cs.query = query
				cs.amfEncoder.EncodeAmf0(cs.conn)
			} else {
				return fmt.Errorf("invalid rtmp parse type tcUrl")
			}
			break
		}
	}

	return nil
}

func (cs *chunkStream) writeChunk(chunkHeader *chunkHeader, chunkData []byte) {

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

func newChunkHeader() *chunkHeader {
	return &chunkHeader{
		basicChunkHeader: &basicChunkHeader{},
		messageHeader:    &messageHeader{},
	}
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

func (rh *rtmpHandler) decodeMessageHeader(buf []byte) error {
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
func (rh *rtmpHandler) decodeFmtType0(buf []byte) error {
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
func (rh *rtmpHandler) decodeFmtType1(buf []byte) error {
	if len(buf) < 7 {
		buf = make([]byte, 7)
	}
	_, err := io.ReadAtLeast(rh.conn, buf, 7)
	if err != nil {
		return err
	}
	mh := rh.chunkHeader.messageHeader
	//stream id no change
	mh.timestampDelta = binary.BigEndian.Uint32(buf[:3])
	mh.messageLen = binary.BigEndian.Uint32(buf[3:6])
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
func (rh *rtmpHandler) decodeFmtType2(buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(rh.conn, buf, 3)
	if err != nil {
		return err
	}
	mh := rh.chunkHeader.messageHeader
	mh.timestampDelta = binary.BigEndian.Uint32(buf[:3])
	mh.timestamp += mh.timestampDelta
	return nil
}
