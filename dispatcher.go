package manconn

import (
	"io"
	"net"
	"sync"
	"time"
)

type Dispatcher struct {
	waiting map[string]waiter
	streams map[string]*stream

	addr string

	mu sync.Mutex
}

func NewDispatcher() *Dispatcher {
	d := Dispatcher{
		waiting: make(map[string]waiter),
		streams: make(map[string]*stream),
		addr:    "localhost:18081",
	}
	go d.accept()
	return &d
}

// goroutine
func (d *Dispatcher) accept() {
	ln, err := net.Listen("tcp", d.addr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("Could not accept: %v.", err)
		}
		go d.waitForMatching(conn)
	}
}

const STREAM_ID_LENGTH = 8
const WAIT_FOR_PAIR_TIME = time.Second * 5

func (d *Dispatcher) waitForMatching(conn net.Conn) {
	buf := make([]byte, STREAM_ID_LENGTH)
	if _, err := io.ReadFull(conn, buf); err == nil {
		sid := string(buf)
		found, waitCh, err := d.findOrSetStreamId(conn, sid)
		if err != nil {
			conn.Close()
		}
		if !found {
			timer := time.NewTimer(WAIT_FOR_PAIR_TIME)
			select {
			case <-waitCh:
				timer.Stop()
			case <-timer.C:
				conn.Write([]byte("timeout"))
				conn.Close()
				return
			}
		}
		conn.Write([]byte("success"))
	} else {
		log.Errorf("Read streamId failed: %v.", err)
	}
}

type waiter struct {
	ch   chan struct{}
	conn net.Conn
}

func (d *Dispatcher) removeStream(sid string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.streams, sid)
	log.Info(d.streams)
}

func (d *Dispatcher) findOrSetStreamId(c net.Conn, sid string) (bool, chan struct{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// if _, found := d.streams[sid]; found == true {
	//     return false, ERR_established_STREAM_ID
	// }
	if w, found := d.waiting[sid]; found == true {
		d.streams[sid] = newStream(sid, d.removeStream, c, w.conn)
		w.ch <- struct{}{}
		delete(d.waiting, sid)
		return true, nil, nil
	} else {
		ch := make(chan struct{})
		d.waiting[sid] = waiter{
			ch:   ch,
			conn: c,
		}
		return false, ch, nil
	}
}

type segment struct {
	data []byte
	due  time.Time
}

type dataSink struct {
	buf       []*segment
	chCanRead chan struct{}

	mu sync.Mutex
}

func newDataSink() *dataSink {
	ds := dataSink{}
	ds.chCanRead = make(chan struct{})
	return &ds
}

func (ds *dataSink) get() *segment {
	for {
		ds.mu.Lock()
		if len(ds.buf) > 0 {
			seg := ds.buf[0]
			ds.buf = ds.buf[1:]
			ds.mu.Unlock()
			return seg
		} else {
			ds.mu.Unlock()
			<-ds.chCanRead
		}
	}
}

func (ds *dataSink) put(seg *segment) {
	ds.mu.Lock()
	ds.buf = append(ds.buf, seg)
	ds.mu.Unlock()
	select {
	case ds.chCanRead <- struct{}{}:
	default:
	}
}

type bandwidthLimitter struct{}

func (bwl *bandwidthLimitter) use(n int) {}

type stream struct {
	sid string

	uplink   *direction
	downlink *direction

	dead chan struct{}

	closed bool
	cb     func(string)
	mu     sync.Mutex
}

func newStream(sid string, cb func(string), c1 net.Conn, c2 net.Conn) *stream {
	st := stream{
		sid:    sid,
		dead:   make(chan struct{}),
		closed: false,
		cb:     cb,
	}
	st.uplink = newDirection(c1, c2, &st)
	st.downlink = newDirection(c2, c1, &st)
	return &st
}

func (s *stream) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.uplink.src.Close()
		s.uplink.dst.Close()
		s.cb(s.sid)
	}
}

type direction struct {
	s *stream

	src net.Conn
	dst net.Conn

	srcUpload   *bandwidthLimitter
	dstDownload *bandwidthLimitter

	delay LatencyAdder

	sink *dataSink
}

func newDirection(src, dst net.Conn, s *stream) *direction {
	di := direction{
		s:           s,
		src:         src,
		dst:         dst,
		srcUpload:   &bandwidthLimitter{},
		dstDownload: &bandwidthLimitter{},
		// delay:       &DummyLatencyAdder{
		//     Latency: time.Millisecond * 100,
		// },
		delay: newTimeRangeLatencyAdder(time.Millisecond, time.Second*3, time.Second),
		sink:  newDataSink(),
	}
	go di.readloop()
	go di.writeloop()
	return &di
}

const MSS = 1500

func (di *direction) readloop() {
	for {
		buf := make([]byte, MSS)
		n, err := di.src.Read(buf)
		if err != nil {
			di.close(err)
			return
		}

		seg := &segment{
			data: buf[:n],
			due:  time.Now().Add(di.delay.Next()),
		}
		// may block
		di.srcUpload.use(len(seg.data))
		di.sink.put(seg)
	}
}

func (di *direction) writeloop() {
	for {
		seg := di.sink.get()
		now := time.Now()
		if seg.due.After(now) {
			time.Sleep(seg.due.Sub(now))
		}
		// may block
		di.dstDownload.use(len(seg.data))
		_, err := di.dst.Write(seg.data)
		if err != nil {
			di.close(err)
			return
		}
	}
}

func (di *direction) close(err error) {
	log.Infof("Stream(%v) met error: %v.", di.s.sid, err)
	di.s.close()
}
