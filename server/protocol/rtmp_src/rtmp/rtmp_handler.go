package rtmp

import (
	"context"
	"fmt"
	"github.com/Opafanls/hylan/server/base"
	"github.com/Opafanls/hylan/server/protocol/container/flv"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/configure"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/session"
	log "github.com/sirupsen/logrus"
	"reflect"
)

func (s *Server) OnStart(ctx context.Context, sess session.HySessionI) (base.StreamBaseI, error) {
	conn0 := sess.GetConn().Conn()
	conn := core.NewConn(conn0, 4*1024)
	s.conn = conn
	if err := conn.HandshakeServer(); err != nil {
		log.Errorf("handle conn handshakeServer err: %+v", err)
		return nil, err
	}
	connServer := core.NewConnServer(conn)
	s.connServer = connServer
	if err := connServer.ReadMsg(); err != nil {
		log.Errorf("handleConn read msg err: %+v", err)
		return nil, err
	}

	appName, name, url := connServer.GetInfo()
	log.Infof("GetInfo: %s, %s, %s", appName, name, url)
	if ret := configure.CheckAppName(appName); !ret {
		err := fmt.Errorf("application name=%s is not configured", appName)
		log.Errorf("CheckAppName err: %+v", err)
		return nil, err
	}

	sInfo, err := base.NewBase(url)
	if err != nil {
		return nil, err
	}

	sInfo.SetParam(base.ParamKeyName, name)
	sInfo.SetParam(base.ParamKeyApp, appName)
	return sInfo, nil
}

func (s *Server) OnStreaming(ctx context.Context, info base.StreamBaseI, sess session.HySessionI) error {
	connServer := s.connServer
	name := info.Name()
	conn := s.conn
	appName := info.App()
	log.Debugf("handleConn: IsPublisher=%v", connServer.IsPublisher())
	if connServer.IsPublisher() {
		if configure.Config.GetBool("rtmp_noauth") {
			key, err := configure.RoomKeys.GetKey(name)
			if err != nil {
				err := fmt.Errorf("Cannot create key err=%s", err.Error())
				conn.Close()
				log.Error("GetKey err: ", err)
				return err
			}
			name = key
		}
		channel, err := configure.RoomKeys.GetChannel(name)
		if err != nil {
			err := fmt.Errorf("invalid key err=%s", err.Error())
			conn.Close()
			log.Error("CheckKey err: ", err)
			return err
		}
		connServer.PublishInfo.Name = channel
		if pushlist, ret := configure.GetStaticPushUrlList(appName); ret && (pushlist != nil) {
			log.Debugf("GetStaticPushUrlList: %v", pushlist)
		}
		reader := NewVirReader(connServer)
		s.handler.HandleReader(reader)
		log.Debugf("new publisher: %+v", reader.Info())

		if s.getter != nil {
			writeType := reflect.TypeOf(s.getter)
			log.Debugf("handleConn:writeType=%v", writeType)
			writer := s.getter.GetWriter(reader.Info())
			s.handler.HandleWriter(writer)
		}
		if configure.Config.GetBool("flv_archive") {
			flvWriter := new(flv.FlvDvr)
			s.handler.HandleWriter(flvWriter.GetWriter(reader.Info()))
		}
	} else {
		writer := NewVirWriter(connServer)
		log.Debugf("new player: %+v", writer.Info())
		s.handler.HandleWriter(writer)
	}

	return nil
}

func (s *Server) OnStop() error {

	s.conn = nil
	s.connServer = nil

	return nil
}
