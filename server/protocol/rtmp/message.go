package rtmp

type Message interface {
	TypeID() TypeID
}

type EncodingType uint8

const (
	EncodingTypeAMF0 EncodingType = 0
	EncodingTypeAMF3 EncodingType = 3
)

type NetConnectionConnectCommand struct {
	App            string       `mapstructure:"app" amf0:"app"`
	Type           string       `mapstructure:"type" amf0:"type"`
	FlashVer       string       `mapstructure:"flashVer" amf0:"flashVer"`
	TCURL          string       `mapstructure:"tcUrl" amf0:"tcUrl"`
	Fpad           bool         `mapstructure:"fpad" amf0:"fpad"`
	Capabilities   int          `mapstructure:"capabilities" amf0:"capabilities"`
	AudioCodecs    int          `mapstructure:"audioCodecs" amf0:"audioCodecs"`
	VideoCodecs    int          `mapstructure:"videoCodecs" amf0:"videoCodecs"`
	VideoFunction  int          `mapstructure:"videoFunction" amf0:"videoFunction"`
	ObjectEncoding EncodingType `mapstructure:"objectEncoding" amf0:"objectEncoding"`
}
