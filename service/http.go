package service

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

type HTTPServer struct {
	Engine *gin.Engine
	Port   int
}

func NewHTTPServer(port int) *HTTPServer {
	return &HTTPServer{
		Engine: gin.Default(),
		Port:   port,
	}
}

func (server HTTPServer) Start() error {
	if err := server.Engine.Run(":" + strconv.Itoa(server.Port)); err != nil {
		return err
	}
	return nil
}

func (server HTTPServer) Stop() error {
	return nil
}
