package base

import (
	"fmt"
	"github.com/Opafanls/hylan/server/constdef"
	"net/url"
	"sync"
	"time"
)

type StreamBaseI interface {
	ID() string
	URL() *url.URL
	GetParam(key string) string
	SetParam(key, value string)
}

type StreamBase struct {
	id          string
	rw          *sync.RWMutex
	url         *url.URL
	onTimestamp int64
	paramMap    map[string]string
}

func NewBase(uri string) (StreamBaseI, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return NewBase0(u), nil
}

func NewBase0(values *url.URL) *StreamBase {
	streamBase := &StreamBase{}
	streamBase.id = id(values)
	streamBase.paramMap = make(map[string]string)
	streamBase.onTimestamp = time.Now().UnixNano()
	streamBase.rw = &sync.RWMutex{}
	streamBase.url = values
	return streamBase
}

func (streamBase *StreamBase) ID() string {
	return streamBase.id
}

func (streamBase *StreamBase) URL() *url.URL {
	return streamBase.url
}

func (streamBase *StreamBase) GetParam(key string) string {
	streamBase.rw.RLock()
	val := streamBase.paramMap[key]
	streamBase.rw.RUnlock()
	return val
}

func (streamBase *StreamBase) SetParam(key, val string) {
	streamBase.rw.Lock()
	streamBase.paramMap[key] = val
	streamBase.rw.Unlock()
}

func id(val *url.URL) string {
	vhost := getPad(val.Host)
	query := val.Query()
	if tmp := query.Get("vhost"); tmp != "" {
		vhost = tmp
	}
	path := getPad(val.Path)
	if tmp := query.Get("path"); tmp != "" {
		path = tmp
	}
	return fmt.Sprintf("%s:%s", vhost, path)
}

func getPad(val string) string {
	if val != "" {
		return val
	}
	return constdef.StreamPad
}
