package bitio

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
)

var (
	WriteTypeErr = errors.New("type is invalid")
)

func WriteBE(writer io.Writer, val interface{}) (int, error) {
	switch val.(type) {
	case uint32:
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, val.(uint32))
		return writer.Write(buf)
	case byte:
		return writer.Write([]byte{val.(byte)})
	case uint16:
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, 2)
		return writer.Write(buf)
	case []byte:
		return writer.Write(val.([]byte))
	}
	return -1, WriteTypeErr
}

func WriteLE(writer io.Writer, val interface{}) (int, error) {
	switch val.(type) {
	case uint32:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, val.(uint32))
		return writer.Write(buf)
	case byte:
		return writer.Write([]byte{val.(byte)})
	case uint16:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, 2)
		return writer.Write(buf)
	case []byte:
		return writer.Write(val.([]byte))
	}
	return -1, WriteTypeErr
}

func WriteUintLE(writer io.Writer, v uint32, n int) (int, error) {
	buf := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b := byte(v) & 0xff
		buf[i] = b
	}
	return writer.Write(buf)
}

func WriteUintBE(writer io.Writer, v uint32, n int) (int, error) {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		b := byte(v>>uint32((n-i-1)<<3)) & 0xff
		buf[i] = b
	}
	return writer.Write(buf)
}

func ReadUintLE(reader io.Reader, n int) (uint32, error) {
	buf := make([]byte, n)
	_, err := io.ReadAtLeast(reader, buf, n)
	if err != nil {
		return 0, err
	}
	ret := uint32(0)
	for i := 0; i < n; i++ {
		b := buf[i]
		ret += uint32(b) << uint32(i*8)
	}
	return ret, nil
}

func ReadUintBE(reader io.Reader, n int) (uint32, error) {
	buf := make([]byte, n)
	_, err := io.ReadAtLeast(reader, buf, n)
	if err != nil {
		return 0, err
	}
	ret := uint32(0)
	for i := 0; i < n; i++ {
		b := buf[i]
		ret = ret<<8 + uint32(b)
	}
	return ret, nil
}

func PutU8(b []byte, v uint8) {
	b[0] = v
}

func PutI16BE(b []byte, v int16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func PutU16BE(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func PutI24BE(b []byte, v int32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

func PutU24BE(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

func PutI32BE(b []byte, v int32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func PutU32BE(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func PutU32LE(b []byte, v uint32) {
	b[3] = byte(v >> 24)
	b[2] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[0] = byte(v)
}

func PutU40BE(b []byte, v uint64) {
	b[0] = byte(v >> 32)
	b[1] = byte(v >> 24)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 8)
	b[4] = byte(v)
}

func PutU48BE(b []byte, v uint64) {
	b[0] = byte(v >> 40)
	b[1] = byte(v >> 32)
	b[2] = byte(v >> 24)
	b[3] = byte(v >> 16)
	b[4] = byte(v >> 8)
	b[5] = byte(v)
}

func PutU64BE(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

func PutI64BE(b []byte, v int64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}
