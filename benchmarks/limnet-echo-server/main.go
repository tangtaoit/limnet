package main

import (
	"flag"
	"fmt"

	"github.com/tangtaoit/limnet"
)

type echoServer struct {
	limnet.DefaultEventHandler
}

func (e *echoServer) OnPacket(c *limnet.Conn, data []byte) {
	c.Write(data)
}

func main() {
	var port int
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.Parse()

	lm := limnet.New(&echoServer{}, limnet.WithAddr(fmt.Sprintf("tcp://127.0.0.1:%d", port)))
	lm.Run()
}
