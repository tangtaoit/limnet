package limnet

import "testing"

type TestHandler struct {
	DefaultEventHandler
}

func (t *TestHandler) OnConnect(c Conn) {
}

func TestServerRun(t *testing.T) {
	s := New(&TestHandler{})
	s.Run()
}
