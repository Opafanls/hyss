package hynet

type NetConfig uint16

const (
	_ = iota
	ReadTimeout
	WriteTimeout
)

type TcpListenConfig struct {
	Addr string
	Port int
}
