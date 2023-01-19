package base

import (
	"encoding/json"
	"fmt"
	"github.com/Opafanls/hylan/server/constdef"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	ParamKeyVhost    = "StreamBaseVhost"
	ParamKeyApp      = "StreamBaseApp"
	ParamKeyName     = "StreamBaseName"
	ParamKeyIsSource = "StreamBaseIsSource"
)

type StreamBaseI interface {
	ID() string
	URL() *url.URL
	GetParam(key string) string
	SetParam(key, value string)
	Vhost() string
	App() string
	Name() string
}

type StreamBase struct {
	id          string
	vhost       string
	app         string
	name        string
	rw          *sync.RWMutex
	url         *url.URL
	onTimestamp int64
	paramMap    map[string]string
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
	//streamBase.id = id(values)
	streamBase.url = values
	return streamBase
}

func newEmpty() *StreamBase {
	streamBase := &StreamBase{}
	streamBase.paramMap = make(map[string]string)
	streamBase.onTimestamp = time.Now().UnixNano()
	streamBase.rw = &sync.RWMutex{}
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
	defer streamBase.format()
	switch key {
	case ParamKeyVhost:
		streamBase.vhost = val
		break
	case ParamKeyApp:
		streamBase.app = val
		break
	case ParamKeyName:
		streamBase.name = val
		break
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

func (streamBase *StreamBase) format() {
	if streamBase.id == "" {
		if streamBase.vhost != "" && streamBase.app != "" && streamBase.name != "" {
			streamBase.id = fmt.Sprintf("%s#%s#%s", streamBase.vhost, streamBase.app, streamBase.name)
		}
	}
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
	return constdef.StreamPad
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
