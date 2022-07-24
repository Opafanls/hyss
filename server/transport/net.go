package transport

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/log"
	"net"
)

type ConnListener struct {
	streamListener net.Listener
}

func (c *ConnListener) TcpListen(ip string, port int, handler func(ctx context.Context, transport Transport)) error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	})
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if err == net.ErrClosed {
				break
			}
			log.Errorf(log.GetEmptyCtx(), "accept conn failed: %+v", err)
			continue
		}
		handler(log.GetCtxWithLogID(), &TcpTransport{Conn: conn})
	}
	return nil
}

func (c *ConnListener) UdpListen(ip string, port int, handler func()) error {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	})
	if err != nil {
		return err
	}
	defer listener.Close()
	//for {
	//
	//}
	return nil
}

func (c *ConnListener) Close(reason string) []error {
	log.Infof(log.GetEmptyCtx(), "close conn listener for reason: %s", reason)
	var errs []error
	if c.streamListener != nil {
		err := c.streamListener.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("close stream_listener failed: %+v", err))
		}
	}
	return errs
}
