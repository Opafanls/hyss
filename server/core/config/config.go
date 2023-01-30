package config

type IConfigKey string

type IConfig interface {
	Init() error
	Load(key string, result interface{}) error
	GetConfig(key IConfigKey) (interface{}, bool)
	SetConfig(key IConfigKey, v interface{}) error
}

//load config from different resource dynamic

type HyConfigCenter struct {
	configCenter map[string]IConfig
	autoFresh    bool
	diffCallback func(configKey string, k, v interface{})
}
