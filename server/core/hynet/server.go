package hynet

import "net"

type ListenServer interface {
	Listen() error
	Init() error
}

type TcpListenServer interface {
	ListenServer
	ConnHandler
}

type UdpListenServer interface {
	ListenServer
	DataHandler
}

type ConnHandler interface {
	HandleConn(conn IHyConn)
}

type DataHandler interface {
	HandleData(data []byte)
}

type TcpServer struct {
	ip   string
	port int

	listener net.Listener
	running  bool

	conn         chan IHyConn
	connChanSize int
}

func NewTcpServer(ip string, port int) *TcpServer {
	s := &TcpServer{}
	s.ip = ip
	s.port = port
	return s
}

func (tcpServer *TcpServer) Listen() error {
	addr := net.TCPAddr{
		IP:   net.ParseIP(tcpServer.ip),
		Port: tcpServer.port,
	}
	listener, err := net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}
	tcpServer.listener = listener
	return nil
}

func (tcpServer *TcpServer) Init() error {
	tcpServer.running = true
	if tcpServer.connChanSize == 0 {
		tcpServer.connChanSize = 1024
	}
	tcpServer.conn = make(chan IHyConn, tcpServer.connChanSize)
	return nil
}

func (tcpServer *TcpServer) Accept() {
	for tcpServer.running {
		conn, err := tcpServer.listener.Accept()
		if err != nil {
			continue
		}
		tcpServer.HandleConn(NewHyConn(conn))
	}
}

func (tcpServer *TcpServer) HandleConn(conn IHyConn) {
	select {
	case tcpServer.conn <- conn:
	default:

	}
}

type UdpServer struct {
	ip   string
	port int

	udpConn *net.UDPConn
	running bool

	dataChan     chan []byte
	dataChanSize int
}

func (u *UdpServer) Listen() error {
	addr := net.UDPAddr{
		Port: u.port,
		IP:   net.ParseIP(u.ip),
	}
	udpConn, err := net.ListenUDP("udp", &addr) // code does not block here
	if err != nil {
		return err
	}
	u.udpConn = udpConn
	return nil
}

func (u *UdpServer) Init() error {
	u.running = true
	if u.dataChanSize == 0 {
		u.dataChanSize = 1024
	}
	u.dataChan = make(chan []byte, u.dataChanSize)
	return nil
}

func (u *UdpServer) HandleData(data []byte) {
	for u.running {
		select {
		case u.dataChan <- data:
		default:

		}
	}
}
