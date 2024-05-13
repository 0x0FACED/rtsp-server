package rtsp

import (
	"fmt"
	"log"
	"net"
	"rtsp-server/internal/rtsp/utils"
	"sync"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

type MediaServerHandler struct {
	s                *gortsplib.Server
	mutex            sync.Mutex
	streams          map[string]*gortsplib.ServerStream
	publishers       map[string]*gortsplib.ServerSession
	StreamManager    *utils.StreamManager
	connMicroservice net.Conn
	URLs             map[int]string
	index            int
}

func (msh *MediaServerHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Printf("User Data: %v", ctx.Conn.NetConn().RemoteAddr())
	log.Printf("conn opened")
}

func (msh *MediaServerHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Printf("conn closed (%v)", ctx.Error)
	log.Printf("conn closed (%v)", ctx.Conn.NetConn().RemoteAddr())
}

func (msh *MediaServerHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Printf("session opened")
}

func (msh *MediaServerHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Println("session closed: ", ctx.Error)
	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	log.Println("session closed: ", ctx.Session.SetuppedPath())
	if msh.streams[ctx.Session.SetuppedPath()] != nil && ctx.Session == msh.publishers[ctx.Session.SetuppedPath()] {
		msh.streams[ctx.Session.SetuppedPath()].Close()
		msh.streams[ctx.Session.SetuppedPath()] = nil
		go func() {
			filename := ctx.Session.SetuppedPath()[11:]
			err := msh.StreamManager.CreateVideo("C:\\videos\\" + filename)
			if err != nil {
				log.Println("Error creating MP4 file:", err)
			}
			ans := make([]byte, 16)
			message := "--file " + filename + ".mp4" + " --model espcn --scale 2\n" +
				"--file " + filename + ".mp4" + " --model edsr --scale 2\n" +
				"--file " + filename + ".mp4" + " --model fsrcnn --scale 2\n" +
				"--file " + filename + ".mp4" + " --model lapsrn --scale 2\n"
			_, err = msh.connMicroservice.Write([]byte(message))
			msh.connMicroservice.Read(ans)
			if err != nil {
				log.Println("Error sending message to Python server:", err)
				return
			}
		}()
	}
}

func (msh *MediaServerHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("describe request")

	msh.mutex.Lock()
	defer msh.mutex.Unlock()

	if msh.streams[ctx.Path] == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, msh.streams[ctx.Path], nil

}

func (msh *MediaServerHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Println("Получен запрос ANNOUNCE на", ctx.Request.URL)
	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	msh.URLs[msh.index] = ctx.Path
	sm := utils.NewStreamManager()
	msh.StreamManager = sm

	msh.streams[ctx.Path] = gortsplib.NewServerStream(msh.s, ctx.Description)
	msh.publishers[ctx.Path] = ctx.Session

	msh.index++
	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (msh *MediaServerHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("setup request")
	msh.mutex.Lock()
	log.Println("SETUP: ", ctx.Path)
	if msh.streams[ctx.Path] == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}
	msh.mutex.Unlock()
	return &base.Response{
		StatusCode: base.StatusOK,
	}, msh.streams[ctx.Path], nil
}

func (msh *MediaServerHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Printf("play request")

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (msh *MediaServerHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Printf("record request")
	path := "C:\\videos\\" + ctx.Path[11:]
	fmt.Println(path)
	writer, err := msh.StreamManager.PrepareWriter(path)

	if err != nil {
		log.Println("Error PrepareWriter(): ", err)
	}
	stream := msh.streams[ctx.Path]
	ctx.Session.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		writer.WriteRTP(pkt)
		stream.WritePacketRTP(medi, pkt)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil

}

func (msh *MediaServerHandler) Server() *gortsplib.Server {
	return msh.s
}

func Setup() *MediaServerHandler {
	streams := make(map[string]*gortsplib.ServerStream)
	publishers := make(map[string]*gortsplib.ServerSession)
	urls := make(map[int]string)
	conn, err := net.Dial("tcp", "0.0.0.0:8081")
	if err != nil {
		log.Fatal("Failed to connect to Python microservice:", err)
	}
	h := &MediaServerHandler{
		streams:          streams,
		publishers:       publishers,
		URLs:             urls,
		index:            0,
		connMicroservice: conn,
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
