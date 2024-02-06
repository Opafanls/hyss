package rtmp

import (
	"context"
	av "github.com/Opafanls/hylan/server/core/av"
	"github.com/Opafanls/hylan/server/protocol/rtmp_src/rtmp/core"
	"github.com/Opafanls/hylan/server/session"
)

type Client struct {
	ctx     context.Context
	handler session.ProtocolHandler
	getter  av.GetWriter
}

func NewRtmpClient(getter av.GetWriter) *Client {
	return &Client{
		getter: getter,
	}
}

func (c *Client) Dial(url string, method string) error {
	connClient := core.NewConnClient()
	if err := connClient.Start(url, method); err != nil {
		return err
	}
	if method == av.PUBLISH {
		//writer := NewVirWriter(connClient)
		//log.Debugf(c.ctx, "client Dial call NewVirWriter url=%s, method=%s", url, method)
		//
	} else if method == av.PLAY {
		//reader := NewVirReader(connClient)
		//log.Debugf(c.ctx, "client Dial call NewVirReader url=%s, method=%s", url, method)
		//c.handler.HandleReader(reader)
		//if c.getter != nil {
		//	writer := c.getter.GetWriter(reader.Info())
		//	c.handler.HandleWriter(writer)
		//}
	}
	return nil
}
