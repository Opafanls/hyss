package pb

import "github.com/aler9/gortsplib/pkg/ringbuffer"

type CacheRing interface {
	Close()
	Pull() (interface{}, bool)
	Push(interface{})
	Reset()
}

type Ring0 struct {
	ringBuffer *ringbuffer.RingBuffer
}

func NewRing0(size uint64) *Ring0 {
	ring := &Ring0{}
	ring.ringBuffer = ringbuffer.New(size)

	return ring
}

func (r *Ring0) Close() {
	r.ringBuffer.Close()
}

func (r *Ring0) Pull() (interface{}, bool) {
	return r.ringBuffer.Pull()
}

func (r *Ring0) Push(data interface{}) {
	r.ringBuffer.Push(data)
}

func (r *Ring0) Reset() {
	r.ringBuffer.Reset()
}
