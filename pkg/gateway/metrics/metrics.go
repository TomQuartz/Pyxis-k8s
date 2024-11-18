package metrics

import (
	"sync/atomic"
	"time"
)

type Throughput struct {
	lastCut    time.Time
	lastMetric float64
	counter    int64
}

func NewThroughput() *Throughput {
	return &Throughput{}
}

func (m *Throughput) Add() {
	atomic.AddInt64(&m.counter, 1)
}

func (m *Throughput) Cut() (float64, float64) {
	now := time.Now()
	elapsed := now.Sub(m.lastCut)
	cnt := atomic.SwapInt64(&m.counter, 0)
	lastMetric := m.lastMetric
	m.lastMetric = float64(cnt) / elapsed.Seconds()
	m.lastCut = now
	return lastMetric, m.lastMetric
}
