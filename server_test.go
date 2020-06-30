package limnet

import "testing"

func TestServerRun(t *testing.T) {
	s := New(&DefaultEventHandler{})
	s.Run()
}
