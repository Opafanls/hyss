package stream

import "sync"

var DefaultHyStreamManager *HyStreamManager

type HyStreamManager struct {
	rwLock    *sync.RWMutex
	streamMap map[string]*HyStream
}

func InitHyStreamManager() {
	DefaultHyStreamManager = NewHyStreamManager()
}

func NewHyStreamManager() *HyStreamManager {
	HyStreamManager := &HyStreamManager{}
	HyStreamManager.rwLock = &sync.RWMutex{}
	HyStreamManager.streamMap = make(map[string]*HyStream)
	return HyStreamManager
}

func (streamManager *HyStreamManager) AddStream(hyStream *HyStream) {
	id := hyStream.StreamBase.ID()
	streamManager.rwLock.Lock()
	streamManager.streamMap[id] = hyStream
	streamManager.rwLock.Unlock()
}

func (streamManager *HyStreamManager) RemoveStream(streamBaseID string) {
	streamManager.rwLock.Lock()
	delete(streamManager.streamMap, streamBaseID)
	streamManager.rwLock.Unlock()
}
