package manconn

import (
	"time"
)

type LatencyAdder interface {
	Next() time.Duration
}

type DummyLatencyAdder struct {
	Latency time.Duration
}

func (la *DummyLatencyAdder) Next() time.Duration {
	return la.Latency
}

type TimeRangeLatencyAdder struct {
	Old   time.Duration
	Range time.Duration
	New   time.Duration

	start time.Time
}

func newTimeRangeLatencyAdder(o, r, n time.Duration) *TimeRangeLatencyAdder {
	la := TimeRangeLatencyAdder{
		Old:   o,
		Range: r,
		New:   n,
		start: time.Now(),
	}
	return &la
}

func (la *TimeRangeLatencyAdder) Next() time.Duration {
	if time.Now().After(la.start.Add(la.Range)) {
		return la.New
	} else {
		return la.Old
	}
}
