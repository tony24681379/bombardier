package lib

import (
	"sync"
	"sync/atomic"
	"time"
)

type completionBarrier interface {
	completed() float64
	tryGrabWork() bool
	jobDone()
	done() <-chan struct{}
	Cancel()
}

type countingCompletionBarrier struct {
	numReqs, reqsGrabbed, reqsDone uint64
	doneChan                       chan struct{}
	closeOnce                      sync.Once
}

func newCountingCompletionBarrier(numReqs uint64) completionBarrier {
	c := new(countingCompletionBarrier)
	c.reqsDone, c.reqsGrabbed, c.numReqs = 0, 0, numReqs
	c.doneChan = make(chan struct{})
	return completionBarrier(c)
}

func (c *countingCompletionBarrier) tryGrabWork() bool {
	select {
	case <-c.doneChan:
		return false
	default:
		reqsDone := atomic.AddUint64(&c.reqsGrabbed, 1)
		return reqsDone <= c.numReqs
	}
}

func (c *countingCompletionBarrier) jobDone() {
	reqsDone := atomic.AddUint64(&c.reqsDone, 1)
	if reqsDone == c.numReqs {
		c.closeOnce.Do(func() {
			close(c.doneChan)
		})
	}
}

func (c *countingCompletionBarrier) done() <-chan struct{} {
	return c.doneChan
}

func (c *countingCompletionBarrier) Cancel() {
	c.closeOnce.Do(func() {
		close(c.doneChan)
	})
}

func (c *countingCompletionBarrier) completed() float64 {
	select {
	case <-c.doneChan:
		return 1.0
	default:
		reqsDone := atomic.LoadUint64(&c.reqsDone)
		return float64(reqsDone) / float64(c.numReqs)
	}
}

type timedCompletionBarrier struct {
	doneChan  chan struct{}
	closeOnce sync.Once
	start     time.Time
	duration  time.Duration
}

func newTimedCompletionBarrier(duration time.Duration) completionBarrier {
	if duration < 0 {
		panic("timedCompletionBarrier: negative duration")
	}
	c := new(timedCompletionBarrier)
	c.doneChan = make(chan struct{})
	c.start = time.Now()
	c.duration = duration
	go func() {
		time.AfterFunc(duration, func() {
			c.closeOnce.Do(func() {
				close(c.doneChan)
			})
		})
	}()
	return completionBarrier(c)
}

func (c *timedCompletionBarrier) tryGrabWork() bool {
	select {
	case <-c.doneChan:
		return false
	default:
		return true
	}
}

func (c *timedCompletionBarrier) jobDone() {
}

func (c *timedCompletionBarrier) done() <-chan struct{} {
	return c.doneChan
}

func (c *timedCompletionBarrier) Cancel() {
	c.closeOnce.Do(func() {
		close(c.doneChan)
	})
}

func (c *timedCompletionBarrier) completed() float64 {
	select {
	case <-c.doneChan:
		return 1.0
	default:
		return float64(time.Since(c.start).Nanoseconds()) /
			float64(c.duration.Nanoseconds())
	}
}
