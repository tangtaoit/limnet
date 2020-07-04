package main

import (
	"flag"
	"fmt"

	"github.com/tangtaoit/limnet"
)

type echoServer struct {
	limnet.DefaultEventHandler
}

func (e *echoServer) OnPacket(c limnet.Conn, data []byte) (out []byte) {
	out = data

	// c.Write(data) TODO： 大量并发 异步write 会有错误
	return
}

func main() {
	var port int
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.Parse()

	lm := limnet.New(&echoServer{}, limnet.WithAddr(fmt.Sprintf("tcp://:%d", port)))
	lm.Run()
}
