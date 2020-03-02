package manconn

import (
	"time"
)

type DelayConfig interface {
	NextDelay() time.Duration
}

type ProbabilityDelay struct {
	// 95%
	// 99%
	// 99.9%
	// 99.99%
	// ...
}

type LongTimeDelay struct {
	base time.Duration
}

func (delay *LongTimeDelay) NextDelay() time.Duration {
	return delay.base
}
