package session

import (
	"github.com/Opafanls/hylan/server/core/cache"
	"github.com/Opafanls/hylan/server/model"
	"time"
)

type HyDataCache struct {
	pktCache cache.CacheRing
	key      cache.CacheRing
	keySent  bool
	firstIdx uint64
}

func NewHyDataCache(size []uint64) *HyDataCache {
	c := &HyDataCache{}
	indexer := []cache.CacheRing{
		c.pktCache,
		c.key,
	}

	for idx, size0 := range size {
		indexer[idx] = cache.NewRing0(size0)
	}

	return c
}

func (cache *HyDataCache) Push(pkt *model.Packet) {
	pktCachedIndex := cache.pktCache.Push(pkt)
	pkt.CacheIdx = pktCachedIndex
	if pkt.IsKeyFrame() {
		cache.key.Push(pkt)
	}
}

func (cache *HyDataCache) Pull() *model.Packet {
	if cache.keySent {
		pkt := cache.blockPull(cache.key)
		cache.firstIdx = pkt.CacheIdx
		cache.pktCache.SetReadIdx(pkt.CacheIdx)
		return pkt
	} else {
		return cache.blockPull(cache.pktCache)
	}
}

func (cache *HyDataCache) blockPull(c cache.CacheRing) *model.Packet {
	for {
		pkt0, exist := c.Pull(false)
		if !exist {
			time.Sleep(time.Millisecond * 300)
			continue
		}
		pkt := pkt0.(*model.Packet)
		return pkt
	}
}
