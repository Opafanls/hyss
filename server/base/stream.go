package base

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type SessionKey string

// {usage}{key_name}
const (
	SessionInitParamKeyVhost      SessionKey = "StreamBaseVhost"
	SessionInitParamKeyApp        SessionKey = "StreamBaseApp"
	SessionInitParamKeyStreamType SessionKey = "StreamBaseStreamType"
	SessionInitParamKeyName       SessionKey = "StreamBaseName"
	SessionInitParamKeyID         SessionKey = "StreamBaseID"

	ConfigKeyStreamBase SessionKey = "ConfigKeyStreamBase"
	ConfigKeyStreamSess SessionKey = "ConfigKeyStreamSess"

	ConfigKeySessionBase SessionKey = "ConfigKeySessionBase"
	ConfigKeyVideoCodec  SessionKey = "ConfigKeyVideoCodec"
	ConfigKeyAudioCodec  SessionKey = "ConfigKeyAudioCodec"
)

type StreamBaseI interface {
	ID() int64
	URL() *url.URL
	GetParam(key SessionKey) (interface{}, bool)
	SetParam(key SessionKey, value interface{})
	Vhost() string
	App() string
	Name() string
}

var _ = StreamBaseI(&StreamBase{})

type StreamBase struct {
	id          int64
	vhost       string
	app         string
	name        string
	rw          *sync.RWMutex
	url         *url.URL
	onTimestamp int64
	paramMap    map[SessionKey]interface{}
}

func NewEmptyBase() *StreamBase {
	return newEmpty()
}

func NewBase(uri string) (StreamBaseI, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	return NewBase0(u), nil
}

func NewBase0(values *url.URL) *StreamBase {
	streamBase := newEmpty()
	streamBase.url = values
	streamBase.vhost = values.Host
	return streamBase
}

func newEmpty() *StreamBase {
	streamBase := &StreamBase{}
	streamBase.id = time.Now().UnixNano()
	streamBase.paramMap = make(map[SessionKey]interface{})
	streamBase.onTimestamp = time.Now().UnixNano()
	streamBase.rw = &sync.RWMutex{}
	return streamBase
}

func (streamBase *StreamBase) ID() int64 {
	return streamBase.id
}

func (streamBase *StreamBase) URL() *url.URL {
	return streamBase.url
}

func (streamBase *StreamBase) GetParam(key SessionKey) (interface{}, bool) {
	streamBase.rw.RLock()
	val, exist := streamBase.paramMap[key]
	streamBase.rw.RUnlock()
	return val, exist
}

func (streamBase *StreamBase) SetParam(key SessionKey, val interface{}) {
	streamBase.rw.Lock()
	streamBase.paramMap[key] = val
	streamBase.rw.Unlock()
	switch key {
	case SessionInitParamKeyVhost:
		streamBase.vhost = val.(string)
	case SessionInitParamKeyApp:
		streamBase.app = val.(string)
	case SessionInitParamKeyName:
		streamBase.name = val.(string)
	case SessionInitParamKeyID:
		streamBase.id = val.(int64)
	default:
		return
	}
}

func (streamBase *StreamBase) Vhost() string {
	return streamBase.vhost
}

func (streamBase *StreamBase) App() string {
	return streamBase.app
}

func (streamBase *StreamBase) Name() string {
	return streamBase.name
}

func id(val *url.URL) string {
	vhost := getPad(val.Host)
	query := val.Query()
	if tmp := query.Get("vhost"); tmp != "" {
		vhost = tmp
	}
	path := getPad(val.Path)
	if tmp := query.Get("app"); tmp != "" {
		path = tmp
	}
	if tmp := query.Get("name"); tmp != "" {
		path = tmp
	}
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return fmt.Sprintf("%s:%s", vhost, path)
}

func getPad(val string) string {
	if val != "" {
		return val
	}
	return StreamPad
}

func (streamBase *StreamBase) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["id"] = streamBase.id
	m["vhost"] = streamBase.vhost
	m["app"] = streamBase.app
	m["name"] = streamBase.name
	m["on_time"] = streamBase.onTimestamp
	m["param"] = streamBase.paramMap
	return json.Marshal(m)
}
