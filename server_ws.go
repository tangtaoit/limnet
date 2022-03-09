package limnet

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// WSServer websocket服务
type WSServer struct {
	limlog.Log
	lnet *LIMNet
	srv  *http.Server
}

// NewWSServer 创建一个websocket服务
func NewWSServer(lnet *LIMNet) *WSServer {
	return &WSServer{
		Log:  limlog.NewLIMLog("WSServer"),
		lnet: lnet,
	}
}

// Start Start
func (s *WSServer) Start() {
	mux := http.NewServeMux()
	s.srv = &http.Server{Addr: s.lnet.opts.WSAddr, Handler: mux}
	mux.HandleFunc("/", s.server)
	go func() {
		var err error
		if s.lnet.opts.SSLOn {
			err = s.srv.ListenAndServe()
		} else {
			err = s.srv.ListenAndServeTLS(s.lnet.opts.SSLCertificate, s.lnet.opts.SSLCertificateKey)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
		s.Debug("Start")
	}()

}

// Stop Stop
func (s *WSServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *WSServer) server(w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		http.NotFound(w, r)
		s.Error("conn creat err", zap.Error(err))
		return
	}
	s.handleNewConnection(conn)
}

func (s *WSServer) handleNewConnection(conn *websocket.Conn) {
	clientID := atomic.AddInt64(&s.lnet.idGen, 1)
	wsconn := NewWSConn(clientID, conn, s.lnet) // 创建一个新的连接
	s.lnet.eventHandler.OnConnect(wsconn)
}
