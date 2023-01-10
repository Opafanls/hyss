package hynet

import (
	"context"
	"github.com/Opafanls/hylan/server/log"
	"github.com/Opafanls/hylan/server/task"
	"net"
)

type ListenServer interface {
	Start() error
	Init() error
	Close()
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
	ctx  context.Context
	ip   string
	port int

	listener net.Listener
	running  bool

	conn         chan IHyConn
	connChanSize int

	ConnHandler ConnHandler

	stop chan struct{}
}

func NewTcpServer(ctx context.Context, ip string, port int) *TcpServer {
	s := &TcpServer{}
	s.ctx = ctx
	s.ip = ip
	s.port = port
	return s
}

func (tcpServer *TcpServer) Start() error {
	addr := net.TCPAddr{
		IP:   net.ParseIP(tcpServer.ip),
		Port: tcpServer.port,
	}
	listener, err := net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}
	tcpServer.listener = listener
	task.SubmitTask0(tcpServer.ctx, func() {
		log.Infof(context.Background(), "listen rtmp server@%s:%d", tcpServer.ip, tcpServer.port)
		tcpServer.Accept()
	})
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
		tcpServer.ConnHandler.HandleConn(NewHyConn(conn))
	}
}

func (tcpServer *TcpServer) Listener() net.Listener {
	return tcpServer.listener
}

func (tcpServer *TcpServer) Close() {
	tcpServer.stop <- struct{}{}
}

type UdpServer struct {
	ip   string
	port int

	udpConn *net.UDPConn
	running bool

	dataChan     chan []byte
	dataChanSize int
}

func (u *UdpServer) Start() error {
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
