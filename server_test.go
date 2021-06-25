package limnet

import (
	"fmt"
	"math"
	"testing"
	"time"
)

type TestHandler struct {
	DefaultEventHandler
}

func (t *TestHandler) OnConnect(c Conn) {
}

func TestServerRun(t *testing.T) {
	s := New(&TestHandler{})
	go s.Run()
	time.Sleep(time.Second)
	defer func() {
		err := s.Stop()
		if err != nil {
			panic(err)
		}
	}()
}

func BenchmarkSendMessage(b *testing.B) {
	b.StopTimer()
	fmt.Println("start......")
	s := New(&TestHandler{})
	go s.Run()
	time.Sleep(time.Millisecond)
	defer func() {
		err := s.Stop()
		if err != nil {
			panic(err)
		}
		fmt.Println("stop......")
	}()
	b.StartTimer()
	for i := 0; i <= b.N; i++ {
		math.Abs(float64(i))
	}
}
