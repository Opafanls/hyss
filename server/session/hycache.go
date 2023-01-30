package session

import (
	"github.com/Opafanls/hylan/server/core/cache"
	"github.com/Opafanls/hylan/server/model"
)

type HyDataCache struct {
	pktCache cache.CacheRing
	key      cache.CacheRing
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
	if pkt.IsKeyFrame() {

	}
}
