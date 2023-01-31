package b0

import "sync"
import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// RingBuffer is a ring buffer.
type RingBuffer struct {
	size       uint64
	readIndex  uint64
	writeIndex uint64
	closed     int64
	buffer     []unsafe.Pointer
	event      *event
}

// New allocates a RingBuffer.
func New(size uint64) (*RingBuffer, error) {
	// when writeIndex overflows, if size is not a power of
	// two, only a portion of the buffer is used.
	if (size & (size - 1)) != 0 {
		return nil, fmt.Errorf("size must be a power of two")
	}

	return &RingBuffer{
		size:       size,
		readIndex:  1,
		writeIndex: 0,
		buffer:     make([]unsafe.Pointer, size),
		event:      newEvent(),
	}, nil
}

// Close makes Pull() return false.
func (r *RingBuffer) Close() {
	atomic.StoreInt64(&r.closed, 1)
	r.event.signal()
}

// Reset restores Pull() behavior after a Close().
func (r *RingBuffer) Reset() {
	for i := uint64(0); i < r.size; i++ {
		atomic.SwapPointer(&r.buffer[i], nil)
	}
	atomic.SwapUint64(&r.writeIndex, 0)
	r.readIndex = 1
	atomic.StoreInt64(&r.closed, 0)
}

// Push pushes data at the end of the buffer.
func (r *RingBuffer) Push(data interface{}) uint64 {
	writeIndex := atomic.AddUint64(&r.writeIndex, 1)
	i := writeIndex % r.size
	atomic.SwapPointer(&r.buffer[i], unsafe.Pointer(&data))
	r.event.signal()
	return i
}

// Pull pulls data from the beginning of the buffer.
func (r *RingBuffer) Pull(sync bool) (interface{}, bool) {
	for {
		i := r.readIndex % r.size
		res := (*interface{})(atomic.SwapPointer(&r.buffer[i], nil))
		if res == nil {
			if atomic.SwapInt64(&r.closed, 0) == 1 {
				return nil, false
			}
			if sync {
				r.event.wait()
				continue
			} else {
				return nil, false
			}
		}

		r.readIndex++
		return *res, true
	}
}

func (r *RingBuffer) SetReadIdx(readIdx uint64) {
	r.readIndex = readIdx
}

func (r *RingBuffer) SetWriteIdx(writeIdx uint64) {
	r.writeIndex = writeIdx
}

type event struct {
	mutex sync.Mutex
	cond  *sync.Cond
	value bool
}

func newEvent() *event {
	cv := &event{}
	cv.cond = sync.NewCond(&cv.mutex)
	return cv
}

func (cv *event) signal() {
	func() {
		cv.mutex.Lock()
		defer cv.mutex.Unlock()
		cv.value = true
	}()

	cv.cond.Broadcast()
}

func (cv *event) wait() {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()

	if !cv.value {
		cv.cond.Wait()
	}

	cv.value = false
}
