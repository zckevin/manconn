package manconn

import (
	"io"
	"net"
	"sync"
)

func DialConnPair(dispatcherAddr string) ([2]net.Conn, error) {
	errCh := make(chan error, 2)
	var conns [2]net.Conn
	var wg sync.WaitGroup

	dialFn := func(sid string, index int) {
		defer wg.Done()
		conn, err := net.Dial("tcp", dispatcherAddr)
		if err != nil {
			log.Errorf("Dial to %s met error: %v", dispatcherAddr, err)
		}
		_, err = conn.Write([]byte(sid))
		if err != nil {
		}
		buf := make([]byte, 7)
		if _, err := io.ReadFull(conn, buf); err != nil {
			errCh <- err
			return
		}
		if string(buf) == "success" {
			conns[index] = conn
		} else {
			log.Error(string(buf))
		}
	}
	sid := randString(8)
	log.Info(sid)
	wg.Add(2)
	go dialFn(sid, 0)
	go dialFn(sid, 1)
	wg.Wait()
	select {
	case err := <-errCh:
		log.Infof("Dial to %v error: %v.", dispatcherAddr, err)
		return [2]net.Conn{}, err
	default:
	}
	return conns, nil
}
