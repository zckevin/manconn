package manconn

import (
	// "io"
	"net"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	d := NewDispatcher()
	time.Sleep(time.Millisecond * 50)
	conns, err := DialConnPair(d.addr)
	if err != nil {
		panic(err)
	}
	log.Info(conns)
	src, dst := conns[0], conns[1]

	f := func(src, dst net.Conn) {
		_, err = src.Write([]byte("123"))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond * 3010)
		_, err = src.Write([]byte("456"))
		if err != nil {
			panic(err)
		}
		// buf := make([]byte, 3)
		// _, err = io.ReadFull(dst, buf)
		buf := make([]byte, 1024*8)
		n, err := dst.Read(buf)
		if err != nil {
			panic(err)
		}
		log.Info(string(buf[:n]))
		buf = make([]byte, 1024*8)
		n, err = dst.Read(buf)
		if err != nil {
			panic(err)
		}
		log.Info(string(buf[:n]))
	}
	f(src, dst)
	// f(dst, src)

	src.Close()

	<-make(chan bool)
}
