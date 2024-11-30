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
	logger           logr.Logger
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

func (p *Pyxis) Schedule(req *workload.ClientRequest) (dest int) {
	x := float64(atomic.LoadInt64(&p.turningPoint)) / PyxisRangeFactor
	taskRange := p.taskBoundary[req.TypeID]
	if x <= taskRange[0] {
		dest = ToStorage
	} else if x >= taskRange[1] {
		dest = ToCompute
	} else {
		if rand.Float64()*(taskRange[1]-taskRange[0]) < x-taskRange[0] {
			dest = ToCompute
		} else {
			dest = ToStorage
		}
	}
	// p.logger.V(1).Info(fmt.Sprintf("Schedule type %d->%d %.2f|[%.2f,%.2f]", req.TypeID, dest, x, taskRange[0], taskRange[1]))
	return dest
}

func (p *Pyxis) Finish(resp *workload.ClientResponse) {
	p.tputMetric.Add()
}

func (p *Pyxis) Run(ctx context.Context) {
	p.logger = log.FromContext(ctx)
	interval := time.Duration(p.cfg.IntervalSecs * float64(time.Second))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.xloop()
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pyxis) xloop() {
	lastTput, Tput := p.tputMetric.Cut()
	lastX, X := p.lastTurningPoint, atomic.LoadInt64(&p.turningPoint)
	p.lastTurningPoint = X
	if lastTput == 0 {
		p.logger.V(1).Info("Xloop skip")
		return
	}
	p.tightenBounds(lastX, X, Tput-lastTput)

	if p.converged ||
		float64(p.upperbound-p.lowerbound)/PyxisRangeFactor < p.cfg.StopPrecision ||
		p.cfg.ReferencePoint >= 0 && math.Abs(float64(X)/PyxisRangeFactor-p.cfg.ReferencePoint) < p.cfg.StopPrecision {
		if !p.converged {
			p.converged = true
			p.logger.Info("Xloop converged", "x", X, "range", fmt.Sprintf("[%d,%d]", p.lowerbound, p.upperbound))
		}
	}

	nextX := X
	if !p.converged {
		step := int64(p.cfg.StepSizeRel * float64(p.upperbound-p.lowerbound))
		direction := float64(X-lastX) * (Tput - lastTput)
		if direction == 0 {
			direction = 0.5*PyxisRangeFactor - float64(X)
		}
		if direction > 0 {
			nextX += step
		} else {
			nextX -= step
		}
		nextX = int64(math.Min(math.Max(float64(nextX), float64(p.lowerbound)), float64(p.upperbound)))
		atomic.StoreInt64(&p.turningPoint, nextX)
	}
	p.logger.V(1).Info("Xloop enter", "converged", p.converged, "x", fmt.Sprintf("%d->%d->%d", lastX, X, nextX), "range", fmt.Sprintf("[%d,%d]", p.lowerbound, p.upperbound), "tput", fmt.Sprintf("%.2fK->%.2fK", lastTput/1000., Tput/1000.))
}

func (p *Pyxis) tightenBounds(lastX, X int64, delta float64) {
	if lastX == X {
		return
	}
	if delta > 0 {
		if X-lastX > 0 {
			p.lowerbound = lastX
		} else {
			p.upperbound = lastX
		}
	} else {
		if X-lastX > 0 {
			p.upperbound = X
		} else {
			p.lowerbound = X
		}
	}
}
