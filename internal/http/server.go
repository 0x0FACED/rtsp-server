package http

import (
	"fmt"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/gin-gonic/gin"
	uuid2 "github.com/google/uuid"
	"log"
	"net"
	"net/http"
	"os"
	"rtsp-server/internal/rtsp"
)

type Server struct {
	mediaServer *rtsp.MediaServerHandler
	router      *gin.Engine
}

func NewServer(msh *rtsp.MediaServerHandler) *Server {
	return &Server{
		mediaServer: msh,
		router:      gin.Default(),
	}
}

func (s *Server) configureRoutes() {
	s.router.POST("/startStream", s.handleStartStream)
	s.router.GET("/setup", s.handeSetupStream)
	s.router.GET("/stream_request", s.handleStreamRequest)
}

func (s *Server) StartServer() {
	s.configureRoutes()
	go func() {
		err := s.router.Run(":8080")
		if err != nil {
			log.Fatal(err)
			return
		}
	}()
	ip, err := externalIP()
	if err != nil {
		fmt.Println("Error during getting ip:", err)
		os.Exit(1)
	}

	log.Printf("HTTP Server started, address: http://%s:%s\n", ip, "8080")
	log.Printf("RTSP Server started, address: rtsp://%s:%s/stream\n", ip, "8554")
	panic(s.mediaServer.Server().StartAndWait())
}

func externalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

func (s *Server) handleStreamRequest(c *gin.Context) {
	streamID, err := uuid2.NewUUID()
	log.Println("Request from: ", c.Request.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"stream_url": streamID.String(),
	})
}

func (s *Server) handleDisconnection() {
	/*
		::GET::
		Здесь конкретно логика отключения пользователя от сервера (не от стрима).
		Этот метод скорее нужен будет для логирования просто

	*/
}

func (s *Server) handeSetupStream(c *gin.Context) {
	session := c.Keys["Session"].(*gortsplib.ServerSession)
	conn := c.Keys["Conn"].(*gortsplib.ServerConn)
	request := c.Keys["Request"].(*base.Request)
	path := c.Keys["Path"].(string)
	query := c.Keys["Query"].(string)
	transport := c.Keys["Transport"].(gortsplib.Transport)
	response, _, err := s.mediaServer.OnSetup(&gortsplib.ServerHandlerOnSetupCtx{
		Session:   session,
		Conn:      conn,
		Request:   request,
		Path:      path,
		Query:     query,
		Transport: transport,
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(int(response.StatusCode), gin.H{"response": response})
}

func (s *Server) handleStartStream(c *gin.Context) {
	uuid, err := uuid2.NewUUID()
	if err != nil {
		return
	}

	streamURL := fmt.Sprintf("/stream/%s", uuid)

	c.JSON(http.StatusOK, gin.H{
		"stream_url": streamURL,
	})

}
