package rtsp

import (
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"net"
	"sync"
)

type MediaServerHandler struct {
	s                *gortsplib.Server
	mutex            sync.Mutex
	stream           *gortsplib.ServerStream
	publisher        *gortsplib.ServerSession
	StreamManager    *StreamManager
	connMicroservice net.Conn
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
	log.Printf("announce request")

	log.Println("Получен запрос ANNOUNCE на ", ctx.Request.URL)

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

	ctx.Session.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		if pkt.PayloadType == 97 {
			_, err := msh.connMicroservice.Write(pkt.Payload)
			if err != nil {
				log.Println("Error during conn.Write(payload): ", err)
			}
		}
		go func() {
			buff := make([]byte, len(pkt.Payload))
			_, err := msh.connMicroservice.Read(buff)
			if err != nil {
				log.Println("Error during conn.Read(buff): ", err)
			}
			pkt.Payload = buff
		}()
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
	h := &MediaServerHandler{}
	err := error(nil)
	h.connMicroservice, err = net.Dial("tcp", "0.0.0.0:8081")
	if err != nil {
		log.Fatal("Failed to connect to Python microservice:", err)
	}
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
