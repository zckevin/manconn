package manconn

import (
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
)

func setLatency(sid, cmdAddr string, latency LatencyAdder) error {
	ch, err := tchannel.NewChannel("cmd-client", nil)
	if err != nil {
		log.Errorf("Could not create channel: %v.", err)
	}
	server := ch.Peers().Add(cmdAddr)

	ctx, cancel := json.NewContext(time.Second * 3)
	defer cancel()

	var resp Response
	config := LatencyConfig{
		Sid:    sid,
		Config: interface{}(latency),
		Type:   reflect.TypeOf(latency).Elem().Name(),
	}
	if err := json.CallPeer(ctx, server, "CommandService", "setLatency", &config, &resp); err != nil {
		return err
	}
	if resp.Success {
		return nil
	} else {
		return errors.New(resp.Err)
	}
}

func DialConnPair(dispatcherAddr, cmdAddr string, latency LatencyAdder) ([2]net.Conn, string, error) {
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
		return [2]net.Conn{}, "", err
	default:
	}

	err := setLatency(sid, cmdAddr, latency)
	if err != nil {
		return [2]net.Conn{}, "", err
	}
	return conns, sid, nil
}
