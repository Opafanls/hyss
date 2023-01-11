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
	buf := make([]byte, n, 0)
	for i := 0; i < n; i++ {
		b := byte(v) & 0xff
		buf[i] = b
		v = v >> 8
	}
	return writer.Write(buf)
}

func WriteUintBE(writer io.Writer, v uint32, n int) (int, error) {
	buf := make([]byte, n, 0)
	for i := 0; i < n; i++ {
		b := byte(v>>uint32((n-i-1)<<3)) & 0xff
		buf[i] = b
		v = v >> 8
	}
	return writer.Write(buf)
}
