package arbiter

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/tomquartz/pyxis-k8s/pkg/gateway/metrics"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const PyxisRangeFactor = 10000.

type Pyxis struct {
	tputMetric       *metrics.Throughput
	turningPoint     int64
	lastTurningPoint int64
	converged        bool
	lowerbound       int64
	upperbound       int64
	taskBoundary     [][]float64
	cfg              *PyxisConfig
}

// assume profile order: compute-intensive -> io-intensive
func NewPyxis(cfg *PyxisConfig) *Pyxis {
	taskBoundary := make([][]float64, len(cfg.TaskProfiles))
	lastBoundary := 0.
	for i, profile := range cfg.TaskProfiles {
		nextBoundary := lastBoundary + profile.Percentage
		taskBoundary[i] = []float64{lastBoundary, nextBoundary}
		lastBoundary = nextBoundary
	}
	return &Pyxis{
		tputMetric:       metrics.NewThroughput(),
		turningPoint:     int64(cfg.StartPoint * PyxisRangeFactor),
		lastTurningPoint: int64(cfg.StartPoint * PyxisRangeFactor),
		lowerbound:       0,
		upperbound:       int64(PyxisRangeFactor),
		taskBoundary:     taskBoundary,
		cfg:              cfg,
	}
}

var _ Arbiter = &Pyxis{}

func (p *Pyxis) Schedule(req *workload.ClientRequest) int {
	x := float64(atomic.LoadInt64(&p.turningPoint)) / PyxisRangeFactor
	taskRange := p.taskBoundary[req.TypeID]
	if x < taskRange[0] {
		return ToStorage
	} else if x >= taskRange[1] {
		return ToCompute
	} else {
		if rand.Float64() < x-taskRange[0] {
			return ToStorage
		} else {
			return ToCompute
		}
	}
}

func (p *Pyxis) Finish(resp *workload.ClientResponse) {
	p.tputMetric.Add()
}

func (p *Pyxis) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	interval := time.Duration(p.cfg.IntervalSecs * float64(time.Second))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.xloop(logger)
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pyxis) xloop(logger logr.Logger) {
	lastTput, Tput := p.tputMetric.Cut()
	lastX, X := p.lastTurningPoint, atomic.LoadInt64(&p.turningPoint)
	p.lastTurningPoint = X
	if lastTput == 0 {
		return
	}
	p.tightenBounds(lastX, X, Tput-lastTput)

	if p.converged ||
		float64(p.upperbound-p.lowerbound)/PyxisRangeFactor < p.cfg.StopPrecision ||
		math.Abs(float64(X)/PyxisRangeFactor-p.cfg.ReferencePoint) < p.cfg.StopPrecision {
		if !p.converged {
			p.converged = true
			logger.Info("Xloop converged", "x", X, "range", fmt.Sprintf("[%d,%d]", p.lowerbound, p.upperbound))
		}
		return
	}

	step := int64(p.cfg.StepSizeRel * float64(p.upperbound-p.lowerbound))
	direction := float64(X-lastX) * (Tput - lastTput)
	if direction == 0 {
		direction = rand.Float64() - 0.5
	}
	nextX := X
	if direction > 0 {
		nextX += step
	} else {
		nextX -= step
	}
	atomic.StoreInt64(&p.turningPoint, nextX)
	logger.V(1).Info("Xloop enter", "x", fmt.Sprintf("%d->%d->%d", lastX, X, nextX), "range", fmt.Sprintf("[%d,%d]", p.lowerbound, p.upperbound), "tput", fmt.Sprintf("%.2fK->%.2fK", lastTput/1000., Tput/1000.))
}

func (p *Pyxis) tightenBounds(lastX, X int64, delta float64) {
	if lastX == X {
		return
	}
	if delta > 0 {
		if X-lastX > 0 {
			p.lowerbound = lastX
		} else {
			p.upperbound = X
		}
	} else {
		if X-lastX > 0 {
			p.upperbound = X
		} else {
			p.lowerbound = lastX
		}
	}
}
