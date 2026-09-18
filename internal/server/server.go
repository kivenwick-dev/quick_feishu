package server

import (
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
	Port   int
}

func New(port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	return &Server{Engine: gin.New(), Port: port}
}

// Listen 监听端口，占用则递增
func (s *Server) Listen() (int, error) {
	port := s.Port
	for i := 0; i < 20; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			port++
			continue
		}
		ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no free port")
}

func (s *Server) Run(port int) error {
	return s.Engine.Run(fmt.Sprintf(":%d", port))
}
