package cache

import (
	"github.com/Opafanls/hylan/server/core/cache/b0"
)

type CacheRing interface {
	Close()
	Pull(sync bool) (interface{}, bool)
	Push(interface{}) uint64
	Reset()
	SetReadIdx(readIdx uint64)
	SetWriteIdx(readIdx uint64)
}

type Ring0 struct {
	ringBuffer *b0.RingBuffer
}

func NewRing0(size uint64) *Ring0 {
	ring := &Ring0{}
	var err error
	ring.ringBuffer, err = b0.New(size)
	if err != nil {
		panic(err)
	}
	return ring
}

func (r *Ring0) Close() {
	r.ringBuffer.Close()
}

func (r *Ring0) Pull(sync bool) (interface{}, bool) {
	return r.ringBuffer.Pull(sync)
}

func (r *Ring0) Push(data interface{}) uint64 {
	return r.ringBuffer.Push(data)
}

func (r *Ring0) Reset() {
	r.ringBuffer.Reset()
}

func (r *Ring0) SetReadIdx(readIdx uint64) {
	r.ringBuffer.SetReadIdx(readIdx)
}
func (r *Ring0) SetWriteIdx(writeIdx uint64) {
	r.ringBuffer.SetWriteIdx(writeIdx)
}
