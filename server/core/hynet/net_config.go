package hynet

type NetConfig uint16

const (
	_ = iota
	ReadTimeout
	WriteTimeout
	RemoteAddr
)

type TcpListenConfig struct {
	Ip   string
	Port int
}

type HttpServeConfig struct {
	Ip   string
	Port int
}
