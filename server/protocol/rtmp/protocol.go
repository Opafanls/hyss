package rtmp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/Opafanls/hylan/server/log"
	"github.com/yutopp/go-amf0"
	"io"
	"time"
)

const controlStreamID = 0
const Version = 3

type TypeID byte

type state string

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
	uninitialized state = "Uninitialized"
	versionSent   state = "versionSent"
	ackSent       state = "ackSent"
	handshakeDone state = "handshakeDone"
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

type chunkStream struct {
	conn      io.ReadWriter
	chunkData *chunkPayload
}

func newChunkStream(conn io.ReadWriter) *chunkStream {
	cs := &chunkStream{}
	cs.conn = conn
	cs.chunkData = &chunkPayload{
		chunkHeader: &chunkHeader{
			basicChunkHeader: &basicChunkHeader{},
			messageHeader:    &messageHeader{},
		},
		chunkData: &chunkData{
			buf: make([]byte, 2048),
		},
	}
	return cs
}

func (cs *chunkStream) decodeChunkStream() error {
	buf := make([]byte, 64)
	err := cs.decodeBasicHeader(nil)
	if err != nil {
		return err
	}
	err = cs.decodeMessageHeader(buf)
	if err != nil {
		return err
	}
	//READ DATA
	cd := cs.chunkData
	chunkData0 := make([]byte, cd.messageLen)
	_, err = io.ReadAtLeast(cs.conn, chunkData0, int(cd.messageLen))
	if err != nil {
		return err
	}
	log.Infof(context.Background(), "message %s", string(chunkData0))
	switch cd.messageTypeID {
	case TypeIDCommandMessageAMF0:
		var object map[string]interface{}
		d := amf0.NewDecoder(bytes.NewReader(chunkData0))
		//netCmd := &NetConnectionConnectCommand{}
		err := d.Decode(&object)
		return err
	}
	return nil
}

/**
+--------------+----------------+--------------------+--------------+
| Basic Header | Message Header | Extended Timestamp |  Chunk Data  |
+--------------+----------------+--------------------+--------------+
|                                                    |
|<------------------- Chunk Header ----------------->|
Chunk Format
*/
type chunkPayload struct {
	*chunkHeader
	*chunkData
}

type chunkHeader struct {
	*basicChunkHeader
	*messageHeader
}

type basicChunkHeader struct { //(1 to 3 bytes)
	fmt  byte
	csID int //[2,65599]
}

type messageHeader struct { //(0, 3, 7, or 11 bytes)
	timestamp       uint32 //3 bytes
	timestampDelta  uint32
	messageLen      uint32 //3 byte
	messageTypeID   TypeID //1 byte
	messageStreamID uint32 //4byte
}

type chunkData struct { //(variable size):
	buf []byte
}

func (cs *chunkStream) decodeBasicHeader(buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(cs.conn, buf[:1], 1)
	if err != nil {
		return err
	}
	basicHeader := cs.chunkData.basicChunkHeader
	basicHeader.fmt = (buf[0] >> 6) & 0b0000_0011
	csID := int(buf[0] & 0b0011_1111)
	switch csID {
	case 0:
		//1 byte
		_, err = io.ReadAtLeast(cs.conn, buf[1:2], 1)
		if err != nil {
			return err
		}
		csID = int(buf[1]) + 64
		break
	case 1:
		//2 bytes
		_, err = io.ReadAtLeast(cs.conn, buf[1:], 2)
		if err != nil {
			return err
		}
		csID = int(buf[2])*256 + int(buf[1]) + 64
		break
	}
	basicHeader.csID = csID
	cs.chunkData.basicChunkHeader = basicHeader
	return nil
}

func (cs *chunkStream) decodeMessageHeader(buf []byte) error {
	fmt0 := cs.chunkData.basicChunkHeader.fmt
	switch fmt0 {
	case 0:
		return cs.decodeFmtType0(buf)
	case 1:
		return cs.decodeFmtType1(buf)
	case 2:
		return cs.decodeFmtType2(buf)
	case 3:
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
func (cs *chunkStream) decodeFmtType0(buf []byte) error {
	if len(buf) < 11 {
		buf = make([]byte, 11)
	}
	_, err := io.ReadAtLeast(cs.conn, buf[:11], 11)
	if err != nil {
		return err
	}
	mh := cs.chunkData.messageHeader
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
		_, err := io.ReadAtLeast(cs.conn, cache32bits, 4)
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
func (cs *chunkStream) decodeFmtType1(buf []byte) error {
	if len(buf) < 7 {
		buf = make([]byte, 7)
	}
	_, err := io.ReadAtLeast(cs.conn, buf, 7)
	if err != nil {
		return err
	}
	mh := cs.chunkData.messageHeader
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
func (cs *chunkStream) decodeFmtType2(buf []byte) error {
	if len(buf) < 3 {
		buf = make([]byte, 3)
	}
	_, err := io.ReadAtLeast(cs.conn, buf, 3)
	if err != nil {
		return err
	}
	mh := cs.chunkData.messageHeader
	mh.timestampDelta = binary.BigEndian.Uint32(buf[:3])
	mh.timestamp += mh.timestampDelta
	return nil
}

func (cs *chunkStream) readChunkData() {
	//cd := cs.chunkData
	//dataLen := cd.messageLen -
}
