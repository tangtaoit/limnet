package limnet

import (
	"net/http"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// WSServer websocket服务
type WSServer struct {
	limlog.Log
	lnet *LIMNet
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
	http.HandleFunc("/", s.server)
	go func() {
		err := http.ListenAndServe(s.lnet.opts.WSAddr, nil)
		if err != nil {
			panic(err)
		}
		s.Debug("启动")
	}()
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
