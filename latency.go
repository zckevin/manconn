package manconn

import (
	"time"
	"math/rand"
)

type LatencyAdder interface {
	Next() time.Duration
}

type DummyLatencyAdder struct {
	Latency time.Duration
}

func addRandomness(t time.Duration) time.Duration {
    return time.Duration(float64(t) * ((rand.Float64() - 0.5) / 5 + 1))
}

func (la *DummyLatencyAdder) Next() time.Duration {
	return addRandomness(la.Latency)
}

type TimeRangeLatencyAdder struct {
	Old   time.Duration
	Range time.Duration
	New   time.Duration

	start time.Time
}

func NewTimeRangeLatencyAdder(o, r, n time.Duration) *TimeRangeLatencyAdder {
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
	    return addRandomness(la.New)
	} else {
	    return addRandomness(la.Old)
	}
}
