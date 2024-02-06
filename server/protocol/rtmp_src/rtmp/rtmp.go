package rtmp

import (
	"context"
	"fmt"
	av2 "github.com/Opafanls/hylan/server/core/av"
	"github.com/Opafanls/hylan/server/protocol/container/flv"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/configure"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/uid"
	"github.com/Opafanls/hylan/server/session"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	maxQueueNum           = 1024
	SAVE_STATICS_INTERVAL = 5000
)

var (
	readTimeout  = configure.Config.GetInt("read_timeout")
	writeTimeout = configure.Config.GetInt("write_timeout")
)

type Client struct {
	handler av2.Handler
	getter  av2.GetWriter
}

func NewRtmpClient(h av2.Handler, getter av2.GetWriter) *Client {
	return &Client{
		handler: h,
		getter:  getter,
	}
}

func (c *Client) Dial(url string, method string) error {
	connClient := core.NewConnClient()
	if err := connClient.Start(url, method); err != nil {
		return err
	}
	if method == av2.PUBLISH {
		writer := NewVirWriter(connClient)
		log.Debugf("client Dial call NewVirWriter url=%s, method=%s", url, method)
		c.handler.HandleWriter(writer)
	} else if method == av2.PLAY {
		reader := NewVirReader(connClient)
		log.Debugf("client Dial call NewVirReader url=%s, method=%s", url, method)
		c.handler.HandleReader(reader)
		if c.getter != nil {
			writer := c.getter.GetWriter(reader.Info())
			c.handler.HandleWriter(writer)
		}
	}
	return nil
}

func (c *Client) GetHandle() av2.Handler {
	return c.handler
}

type Server struct {
	handler av2.Handler
	getter  av2.GetWriter

	connServer *core.ConnServer
	conn       *core.Conn

	sessionHandler session.ProtocolHandler
}

func NewRtmpServer(h av2.Handler, getter av2.GetWriter) *Server {
	return &Server{
		handler: h,
		getter:  getter,
	}
}

func (s *Server) Serve(listener net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp serve panic: ", r)
		}
	}()

	for {
		var netconn net.Conn
		netconn, err = listener.Accept()
		if err != nil {
			return
		}
		conn := core.NewConn(netconn, 4*1024)
		log.Debug("new client, connect remote: ", conn.RemoteAddr().String(),
			"local:", conn.LocalAddr().String())
		go func() {
			err := s.handleConn(conn)
			log.Infof("handleConn done with err: %+v", err)
			if err != nil {
				log.Errorf("handleConn err with close err: %+v", conn.Close())
				return
			}
		}()
	}
}

func (s *Server) handleConn(conn *core.Conn) error {
	err := s.connInit(conn)
	if err != nil {
		return err
	}
	err = s.oncheck()
	if err != nil {
		return err
	}
	return s.handleServerConn(context.Background())
}

func (s *Server) connInit(conn *core.Conn) error {
	if err := conn.HandshakeServer(); err != nil {
		log.Error("handleConn HandshakeServer err: ", err)
		return err
	}
	connServer := core.NewConnServer(conn)
	s.connServer = connServer
	return nil
}

func (s *Server) oncheck() error {
	connServer := s.connServer
	if err := connServer.ReadMsg(); err != nil {
		log.Error("handleConn read msg err: ", err)
		return err
	}

	appname, name, rtmpUrl := connServer.GetInfo()
	connServer.Appname = appname
	connServer.Name = name
	connServer.Url = rtmpUrl
	if ret := configure.CheckAppName(appname); !ret {
		err := fmt.Errorf("application name=%s is not configured", appname)
		log.Error("CheckAppName err: ", err)
		return err
	}

	return nil
}

func (s *Server) handleServerConn(ctx context.Context) error {
	connServer := s.connServer
	log.Debugf("handleConn: IsPublisher=%v", connServer.IsPublisher())
	if connServer.IsPublisher() {
		return s.Publish(ctx, nil)
	} else {
		return s.OnSink(ctx, nil)
	}
}

func (s *Server) OnSink(ctx context.Context, info *session.SinkArg) error {
	writer := NewVirWriter(s.connServer)
	log.Debugf("new player: %+v", writer.Info())
	s.handler.HandleWriter(writer)
	return nil
}

func (s *Server) Publish(ctx context.Context, arg *session.SourceArg) error {
	var (
		connServer = s.connServer
		appname    = connServer.Appname
		name       = connServer.Name
	)
	if configure.Config.GetBool("rtmp_noauth") {
		key, err := configure.RoomKeys.GetKey(name)
		if err != nil {
			err := fmt.Errorf("Cannot create key err=%s", err.Error())
			log.Error("GetKey err: ", err)
			return err
		}
		name = key
	}
	channel, err := configure.RoomKeys.GetChannel(name)
	if err != nil {
		err := fmt.Errorf("invalid key err=%s", err.Error())
		log.Error("CheckKey err: ", err)
		return err
	}
	connServer.PublishInfo.Name = channel
	if pushlist, ret := configure.GetStaticPushUrlList(appname); ret && (pushlist != nil) {
		log.Debugf("GetStaticPushUrlList: %v", pushlist)
	}
	reader := NewVirReader(connServer)
	s.handler.HandleReader(reader)
	//if s.getter != nil {
	//	writeType := reflect.TypeOf(s.getter)
	//	log.Debugf("handleConn:writeType=%v", writeType)
	//	writer := s.getter.GetWriter(reader.Info())
	//	s.handler.HandleWriter(writer)
	//}
	//if configure.Config.GetBool("flv_archive") {
	//	flvWriter := new(flv.FlvDvr)
	//	s.handler.HandleWriter(flvWriter.GetWriter(reader.Info()))
	//}
	log.Infof("new publisher: %+v with getter: %v, flv_archive: %v", reader.Info(), s.getter, configure.Config.GetBool("flv_archive"))
	return nil
}

type GetInFo interface {
	GetInfo() (string, string, string)
}

type StreamReadWriteCloser interface {
	GetInFo
	Close(error)
	Write(core.ChunkStream) error
	Read(c *core.ChunkStream) error
}

type StaticsBW struct {
	StreamId               uint32
	VideoDatainBytes       uint64
	LastVideoDatainBytes   uint64
	VideoSpeedInBytesperMS uint64

	AudioDatainBytes       uint64
	LastAudioDatainBytes   uint64
	AudioSpeedInBytesperMS uint64

	LastTimestamp int64
}

type VirWriter struct {
	Uid    string
	closed bool
	av2.RWBaser
	conn        StreamReadWriteCloser
	packetQueue chan *av2.Packet
	WriteBWInfo StaticsBW
}

func NewVirWriter(conn StreamReadWriteCloser) *VirWriter {
	ret := &VirWriter{
		Uid:         uid.NewId(),
		conn:        conn,
		RWBaser:     av2.NewRWBaser(time.Second * time.Duration(writeTimeout)),
		packetQueue: make(chan *av2.Packet, maxQueueNum),
		WriteBWInfo: StaticsBW{0, 0, 0, 0, 0, 0, 0, 0},
	}

	go ret.Check()
	go func() {
		err := ret.SendPacket()
		if err != nil {
			log.Warning(err)
		}
	}()
	return ret
}

func (v *VirWriter) SaveStatics(streamid uint32, length uint64, isVideoFlag bool) {
	nowInMS := int64(time.Now().UnixNano() / 1e6)

	v.WriteBWInfo.StreamId = streamid
	if isVideoFlag {
		v.WriteBWInfo.VideoDatainBytes = v.WriteBWInfo.VideoDatainBytes + length
	} else {
		v.WriteBWInfo.AudioDatainBytes = v.WriteBWInfo.AudioDatainBytes + length
	}

	if v.WriteBWInfo.LastTimestamp == 0 {
		v.WriteBWInfo.LastTimestamp = nowInMS
	} else if (nowInMS - v.WriteBWInfo.LastTimestamp) >= SAVE_STATICS_INTERVAL {
		diffTimestamp := (nowInMS - v.WriteBWInfo.LastTimestamp) / 1000

		v.WriteBWInfo.VideoSpeedInBytesperMS = (v.WriteBWInfo.VideoDatainBytes - v.WriteBWInfo.LastVideoDatainBytes) * 8 / uint64(diffTimestamp) / 1000
		v.WriteBWInfo.AudioSpeedInBytesperMS = (v.WriteBWInfo.AudioDatainBytes - v.WriteBWInfo.LastAudioDatainBytes) * 8 / uint64(diffTimestamp) / 1000

		v.WriteBWInfo.LastVideoDatainBytes = v.WriteBWInfo.VideoDatainBytes
		v.WriteBWInfo.LastAudioDatainBytes = v.WriteBWInfo.AudioDatainBytes
		v.WriteBWInfo.LastTimestamp = nowInMS
	}
}

func (v *VirWriter) Check() {
	var c core.ChunkStream
	for {
		if err := v.conn.Read(&c); err != nil {
			v.Close(err)
			return
		}
	}
}

func (v *VirWriter) DropPacket(pktQue chan *av2.Packet, info av2.Info) {
	log.Warningf("[%v] packet queue max!!!", info)
	for i := 0; i < maxQueueNum-84; i++ {
		tmpPkt, ok := <-pktQue
		// try to don't drop audio
		if ok && tmpPkt.IsAudio {
			if len(pktQue) > maxQueueNum-2 {
				log.Debug("drop audio pkt")
				<-pktQue
			} else {
				pktQue <- tmpPkt
			}

		}

		if ok && tmpPkt.IsVideo {
			videoPkt, ok := tmpPkt.Header.(av2.VideoPacketHeader)
			// dont't drop sps config and dont't drop key frame
			if ok && (videoPkt.IsSeq() || videoPkt.IsKeyFrame()) {
				pktQue <- tmpPkt
			}
			if len(pktQue) > maxQueueNum-10 {
				log.Debug("drop video pkt")
				<-pktQue
			}
		}

	}
	log.Debug("packet queue len: ", len(pktQue))
}

func (v *VirWriter) Write(p *av2.Packet) (err error) {
	err = nil
	if v.closed {
		err = fmt.Errorf("VirWriter closed")
		return
	}
	if len(v.packetQueue) >= maxQueueNum-24 {
		v.DropPacket(v.packetQueue, v.Info())
	} else {
		v.packetQueue <- p
	}

	return
}

func (v *VirWriter) SendPacket() error {
	Flush := reflect.ValueOf(v.conn).MethodByName("Flush")
	var cs core.ChunkStream
	for {
		p, ok := <-v.packetQueue
		if ok {
			cs.Data = p.Data
			cs.Length = uint32(len(p.Data))
			cs.StreamID = p.StreamID
			cs.Timestamp = p.TimeStamp
			cs.Timestamp += v.BaseTimeStamp()

			if p.IsVideo {
				cs.TypeID = av2.TAG_VIDEO
			} else {
				if p.IsMetadata {
					cs.TypeID = av2.TAG_SCRIPTDATAAMF0
				} else {
					cs.TypeID = av2.TAG_AUDIO
				}
			}

			v.SaveStatics(p.StreamID, uint64(cs.Length), p.IsVideo)
			v.SetPreTime()
			v.RecTimeStamp(cs.Timestamp, cs.TypeID)
			err := v.conn.Write(cs)
			if err != nil {
				v.closed = true
				return err
			}
			Flush.Call(nil)
		} else {
			return fmt.Errorf("closed")
		}

	}
}

func (v *VirWriter) Info() (ret av2.Info) {
	ret.UID = v.Uid
	_, _, URL := v.conn.GetInfo()
	ret.URL = URL
	_url, err := url.Parse(URL)
	if err != nil {
		log.Warning(err)
	}
	ret.Key = strings.TrimLeft(_url.Path, "/")
	ret.Inter = true
	return
}

func (v *VirWriter) Close(err error) {
	log.Warning("player ", v.Info(), "closed: "+err.Error())
	if !v.closed {
		close(v.packetQueue)
	}
	v.closed = true
	v.conn.Close(err)
}

type VirReader struct {
	Uid string
	av2.RWBaser
	demuxer    *flv.Demuxer
	conn       StreamReadWriteCloser
	ReadBWInfo StaticsBW
}

func NewVirReader(conn StreamReadWriteCloser) *VirReader {
	return &VirReader{
		Uid:        uid.NewId(),
		conn:       conn,
		RWBaser:    av2.NewRWBaser(time.Second * time.Duration(writeTimeout)),
		demuxer:    flv.NewDemuxer(),
		ReadBWInfo: StaticsBW{0, 0, 0, 0, 0, 0, 0, 0},
	}
}

func (v *VirReader) SaveStatics(streamid uint32, length uint64, isVideoFlag bool) {
	nowInMS := int64(time.Now().UnixNano() / 1e6)

	v.ReadBWInfo.StreamId = streamid
	if isVideoFlag {
		v.ReadBWInfo.VideoDatainBytes = v.ReadBWInfo.VideoDatainBytes + length
	} else {
		v.ReadBWInfo.AudioDatainBytes = v.ReadBWInfo.AudioDatainBytes + length
	}

	if v.ReadBWInfo.LastTimestamp == 0 {
		v.ReadBWInfo.LastTimestamp = nowInMS
	} else if (nowInMS - v.ReadBWInfo.LastTimestamp) >= SAVE_STATICS_INTERVAL {
		diffTimestamp := (nowInMS - v.ReadBWInfo.LastTimestamp) / 1000

		//log.Printf("now=%d, last=%d, diff=%d", nowInMS, v.ReadBWInfo.LastTimestamp, diffTimestamp)
		v.ReadBWInfo.VideoSpeedInBytesperMS = (v.ReadBWInfo.VideoDatainBytes - v.ReadBWInfo.LastVideoDatainBytes) * 8 / uint64(diffTimestamp) / 1000
		v.ReadBWInfo.AudioSpeedInBytesperMS = (v.ReadBWInfo.AudioDatainBytes - v.ReadBWInfo.LastAudioDatainBytes) * 8 / uint64(diffTimestamp) / 1000

		v.ReadBWInfo.LastVideoDatainBytes = v.ReadBWInfo.VideoDatainBytes
		v.ReadBWInfo.LastAudioDatainBytes = v.ReadBWInfo.AudioDatainBytes
		v.ReadBWInfo.LastTimestamp = nowInMS
	}
}

func (v *VirReader) Read(p *av2.Packet) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Warning("rtmp read packet panic: ", r)
		}
	}()

	v.SetPreTime()
	var cs core.ChunkStream
	for {
		err = v.conn.Read(&cs)
		if err != nil {
			return err
		}
		if cs.TypeID == av2.TAG_AUDIO ||
			cs.TypeID == av2.TAG_VIDEO ||
			cs.TypeID == av2.TAG_SCRIPTDATAAMF0 ||
			cs.TypeID == av2.TAG_SCRIPTDATAAMF3 {
			break
		} else {
			log.Infof("invalid tag: %d", cs.TypeID)
		}
	}

	p.IsAudio = cs.TypeID == av2.TAG_AUDIO
	p.IsVideo = cs.TypeID == av2.TAG_VIDEO
	p.IsMetadata = cs.TypeID == av2.TAG_SCRIPTDATAAMF0 || cs.TypeID == av2.TAG_SCRIPTDATAAMF3
	p.StreamID = cs.StreamID
	p.Data = cs.Data
	p.TimeStamp = cs.Timestamp

	v.SaveStatics(p.StreamID, uint64(len(p.Data)), p.IsVideo)
	v.demuxer.DemuxH(p)
	return err
}

func (v *VirReader) Info() (ret av2.Info) {
	ret.UID = v.Uid
	_, _, URL := v.conn.GetInfo()
	ret.URL = URL
	_url, err := url.Parse(URL)
	if err != nil {
		log.Warning(err)
	}
	ret.Key = strings.TrimLeft(_url.Path, "/")
	return
}

func (v *VirReader) Close(err error) {
	log.Debug("publisher ", v.Info(), "closed: "+err.Error())
	v.conn.Close(err)
}
