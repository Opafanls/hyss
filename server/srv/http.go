package srv

import (
	"fmt"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/stream"
	"github.com/gin-gonic/gin"
)

const (
	msgSuccess     = "success"
	msgArgErr      = "arg_err"
	msgInternalErr = "internal_err"
)

func NewHttpServer(httpServeConfig *hynet.HttpServeConfig) *HttpServer {
	return &HttpServer{
		addr: httpServeConfig.Ip,
		port: httpServeConfig.Port,
	}
}

type HttpServer struct {
	port int
	addr string

	e *gin.Engine
}

func (h *HttpServer) Name() string {
	return "http_api"
}

func (h *HttpServer) Serve(tag string) error {
	return h.e.Run(fmt.Sprintf("%s:%d", h.addr, h.port))
}

func (h *HttpServer) Init(ch hynet.ConnHandler) error {
	e := gin.Default()
	h.e = e
	v1 := e.Group("v1")
	h.v1(v1)
	return nil
}

func (h *HttpServer) Close() {
	return
}

func (h *HttpServer) v1(v1 *gin.RouterGroup) {
	streamG := v1.Group("stream")
	{
		streamG.GET("list", h.listStreams)
	}
}

func (h *HttpServer) listStreams(c *gin.Context) {
	r, _ := stream.DefaultHyStreamManager.StreamFilter(nil)
	h.ok(c, r)
}

func (h *HttpServer) ok(c *gin.Context, data interface{}) {
	h.ret(c, 200, msgSuccess, data)
}

func (h *HttpServer) ret(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(code, gin.H{
		"message":     msg,
		"data_format": data,
	})
}
