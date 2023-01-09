package rtmp

import (
	"github.com/Opafanls/hylan/server/protocol"
	"github.com/sirupsen/logrus"
)

// ControlStreamID StreamID 0 is a control stream
const ControlStreamID = 0

type RtmpStreamType int

const (
	streamStateUnknown RtmpStreamType = iota
	streamStateServerNotConnected
	streamStateServerConnected
	streamStateServerInactive
	streamStateServerPublish
	streamStateServerPlay
	streamStateClientNotConnected
	streamStateClientConnected
)

func (s RtmpStreamType) String() string {
	switch s {
	case streamStateServerNotConnected:
		return "NotConnected(Server)"
	case streamStateServerConnected:
		return "Connected(Server)"
	case streamStateServerInactive:
		return "Inactive(Server)"
	case streamStateServerPublish:
		return "Publish(Server)"
	case streamStateServerPlay:
		return "Play(Server)"
	case streamStateClientNotConnected:
		return "NotConnected(Client)"
	case streamStateClientConnected:
		return "Connected(Client)"
	default:
		return "<Unknown>"
	}
}

type Config struct {
	Handler                   protocol.Handler
	SkipHandshakeVerification bool

	IgnoreMessagesOnNotExistStream          bool
	IgnoreMessagesOnNotExistStreamThreshold uint32

	ReaderBufferSize int
	WriterBufferSize int

	ControlState StreamControlStateConfig

	Logger  logrus.FieldLogger
	RPreset ResponsePreset
}

type StreamControlStateConfig struct {
	DefaultChunkSize uint32
	MaxChunkSize     uint32
	MaxChunkStreams  int

	DefaultAckWindowSize int32
	MaxAckWindowSize     int32

	DefaultBandwidthWindowSize int32
	DefaultBandwidthLimitType  LimitType
	MaxBandwidthWindowSize     int32

	MaxMessageSize    uint32
	MaxMessageStreams int
}

type LimitType uint8

const (
	LimitTypeHard LimitType = iota
	LimitTypeSoft
	LimitTypeDynamic
)
