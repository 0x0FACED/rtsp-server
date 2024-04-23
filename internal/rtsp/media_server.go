package rtsp

import (
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"sync"
)

type MediaServerHandler struct {
	s         *gortsplib.Server
	mutex     sync.Mutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession // публикующий (текущая сессия)
}

// OnConnOpen called when a connection is opened.
func (msh *MediaServerHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Printf("User Data: %v", ctx.Conn.NetConn().RemoteAddr())
	log.Printf("conn opened")
}

// OnConnClose called when a connection is closed.
func (msh *MediaServerHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Printf("conn closed (%v)", ctx.Error)
}

// OnSessionOpen called when a session is opened.
func (msh *MediaServerHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Printf("session opened")
}

// OnSessionClose called when a session is closed.
func (msh *MediaServerHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Printf("session closed")

	msh.mutex.Lock()
	defer msh.mutex.Unlock()

	// if the session is the publisher,
	// close the stream and disconnect any reader.
	if msh.stream != nil && ctx.Session == msh.publisher {
		msh.stream.Close()
		msh.stream = nil
	}
}

// OnDescribe called when receiving a DESCRIBE request.
func (msh *MediaServerHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("describe request")

	msh.mutex.Lock()
	defer msh.mutex.Unlock()

	// no one is publishing yet
	if msh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send medias that are being published to the client
	return &base.Response{
		StatusCode: base.StatusOK,
	}, msh.stream, nil

}

// OnAnnounce called when receiving an ANNOUNCE request.
func (msh *MediaServerHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Printf("announce request")

	// Обработка запроса ANNOUNCE для пути /app/stream1
	log.Println("Получен запрос ANNOUNCE на ", ctx.Request.URL)
	// Ваша логика обработки здесь...

	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	// disconnect existing publisher
	if msh.stream != nil {
		msh.stream.Close()
		msh.publisher.Close()
	}

	// create the stream and save the publisher
	msh.stream = gortsplib.NewServerStream(msh.s, ctx.Description)
	msh.publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

// OnSetup called when receiving a SETUP request.
func (msh *MediaServerHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("setup request")

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

// OnPlay called when receiving a PLAY request.
func (msh *MediaServerHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Printf("play request")

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

// OnRecord called when receiving a RECORD request.
func (msh *MediaServerHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Printf("record request")

	// called when receiving a RTP packet
	ctx.Session.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		// route the RTP packet to all readers
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
	// configure the server
	h := &MediaServerHandler{}

	h.s = &gortsplib.Server{
		Handler:           h,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	// setupAndStart server and wait until a fatal error
	log.Printf("server is ready")
	//panic(h.s.StartAndWait())
	return h
}
