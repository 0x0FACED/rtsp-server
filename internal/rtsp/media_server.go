package rtsp

import (
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"net"
	"rtsp-server/internal/rtsp/utils"
	"sync"
)

type MediaServerHandler struct {
	s                *gortsplib.Server
	mutex            sync.Mutex
	stream           *gortsplib.ServerStream
	publisher        *gortsplib.ServerSession
	StreamManager    *utils.StreamManager
	connMicroservice net.Conn
	URL              string
}

func (msh *MediaServerHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Printf("User Data: %v", ctx.Conn.NetConn().RemoteAddr())
	log.Printf("conn opened")
}

func (msh *MediaServerHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Printf("conn closed (%v)", ctx.Error)
}

func (msh *MediaServerHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Printf("session opened")
}

func (msh *MediaServerHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Printf("session closed")
	msh.mutex.Lock()
	defer msh.mutex.Unlock()

	if msh.stream != nil && ctx.Session == msh.publisher {
		msh.stream.Close()
		msh.stream = nil

		if msh.StreamManager.H264Writer != nil {
			err := msh.StreamManager.H264Writer.Close()
			if err != nil {
				log.Println("Error closing H264 writer:", err)
			}
		}
		go func() {
			err := msh.StreamManager.CreateVideo()
			if err != nil {
				log.Println("Error creating MP4 file:", err)
			}
		}()
	}
}

func (msh *MediaServerHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("describe request")

	msh.mutex.Lock()
	defer msh.mutex.Unlock()

	if msh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, msh.stream, nil

}

func (msh *MediaServerHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Println("Получен запрос ANNOUNCE на", ctx.Request.URL)
	msh.URL = ctx.Request.URL.String()

	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	if msh.stream != nil {
		msh.stream.Close()
		msh.publisher.Close()
	}

	msh.stream = gortsplib.NewServerStream(msh.s, ctx.Description)
	msh.publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (msh *MediaServerHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("setup request")
	log.Println("SETUP: ", ctx.Path)
	// no one is publishing yet
	if msh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, msh.stream, nil
}

func (msh *MediaServerHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Printf("play request")

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func readFromMicroservice(conn net.Conn) {
	defer conn.Close()

	for {
		buff := make([]byte, 1024)
		n, err := conn.Read(buff)
		if err != nil {
			log.Println("Error reading data from Python microservice:", err)
			return
		}

		log.Println("Received data from Python microservice:", buff[:n])
	}
}

func (msh *MediaServerHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Printf("record request")
	err := error(nil)
	msh.StreamManager.H264Writer, err = msh.StreamManager.PrepareWriter()
	if err != nil {
		log.Println("Error PrepareWriter(): ", err)
	}
	ctx.Session.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		msh.StreamManager.H264Writer.WriteRTP(pkt)
		msh.stream.WritePacketRTP(medi, pkt)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil

}

func (msh *MediaServerHandler) Server() *gortsplib.Server {
	return msh.s
}

func Setup() *MediaServerHandler {
	sm := utils.NewStreamManager()
	h := &MediaServerHandler{
		StreamManager: sm,
	}
	//err := error(nil)
	//h.connMicroservice, err = net.Dial("tcp", "0.0.0.0:8081")
	//if err != nil {
	//	log.Fatal("Failed to connect to Python microservice:", err)
	//}
	h.s = &gortsplib.Server{
		Handler:           h,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	log.Printf("server is ready")
	return h
}
