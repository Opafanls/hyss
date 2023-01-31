package srv

import (
	"fmt"
	"github.com/Opafanls/hylan/server/core/hynet"
	"github.com/Opafanls/hylan/server/stream"
	"github.com/gin-gonic/gin"
	"net/http"
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
	var r map[string]stream.HyStreamI
	err := stream.DefaultHyStreamManager.StreamFilter(func(m map[string]stream.HyStreamI) {
		r = m
	})
	if err != nil {
		h.internalErr(c, err)
		return
	}
	h.ok(c, r)
}

func (h *HttpServer) ok(c *gin.Context, data interface{}) {
	h.ret(c, 200, msgSuccess, data, nil)
}

func (h *HttpServer) internalErr(c *gin.Context, err error) {
	h.ret(c, 500, msgInternalErr, nil, err)
}

func (h *HttpServer) argErr(c *gin.Context, err error) {
	h.ret(c, 400, msgArgErr, nil, err)
}

func (h *HttpServer) ret(c *gin.Context, code int, msg string, data interface{}, err error) {
	if code == http.StatusOK {
		c.JSON(code, gin.H{
			"message": msg,
			"data":    data,
		})
	} else {
		c.JSON(code, gin.H{
			"message": msg,
			"err":     err,
		})
	}
}
