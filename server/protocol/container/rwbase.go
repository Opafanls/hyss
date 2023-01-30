package container

import (
	"github.com/Opafanls/hylan/server/model"
	"sync"
	"time"
)

const (
	Video = 11
	Audio = 12
)

type RWBaser struct {
	lock               sync.Mutex
	timeout            time.Duration
	PreTime            time.Time
	BaseTimestamp      uint32
	LastVideoTimestamp uint32
	LastAudioTimestamp uint32
}

func NewRWBaser(duration time.Duration) *RWBaser {
	return &RWBaser{
		timeout: duration,
		PreTime: time.Now(),
	}
}

func (rw *RWBaser) BaseTimeStamp() uint32 {
	return rw.BaseTimestamp
}

func (rw *RWBaser) CalcBaseTimestamp() {
	if rw.LastAudioTimestamp > rw.LastVideoTimestamp {
		rw.BaseTimestamp = rw.LastAudioTimestamp
	} else {
		rw.BaseTimestamp = rw.LastVideoTimestamp
	}
}

func (rw *RWBaser) RecTimeStamp(timestamp, typeID uint32) {
	if typeID == Video {
		rw.LastVideoTimestamp = timestamp
	} else if typeID == Audio {
		rw.LastAudioTimestamp = timestamp
	}
}

func (rw *RWBaser) SetPreTime() {
	rw.lock.Lock()
	rw.PreTime = time.Now()
	rw.lock.Unlock()
}

func (rw *RWBaser) Alive() bool {
	rw.lock.Lock()
	b := !(time.Now().Sub(rw.PreTime) >= rw.timeout)
	rw.lock.Unlock()
	return b
}

func (rw *RWBaser) Init() error {
	//TODO implement me
	panic("implement me")
}

func (rw *RWBaser) Read(packet *model.Packet) error {
	//TODO implement me
	panic("implement me")
}

func (rw *RWBaser) Write(packet *model.Packet) error {
	//TODO implement me
	panic("implement me")
}

func (rw *RWBaser) OnError(err error) bool {
	return false
}
