package arbiter

import (
	"context"
	"math/rand"

	"github.com/tomquartz/pyxis-k8s/pkg/workload"
)

type Kayak struct {
	x   float64
	cfg *KayakConfig
}

// assume profile order: compute-intensive -> io-intensive
func NewKayak(cfg *KayakConfig) *Kayak {
	return &Kayak{
		x:   cfg.StartPoint,
		cfg: cfg,
	}
}

var _ Arbiter = &Kayak{}

func (k *Kayak) Schedule(req *workload.ClientRequest) int {
	if rand.Float64() < k.x {
		return ToCompute
	} else {
		return ToStorage
	}
}

func (k *Kayak) Finish(resp *workload.ClientResponse) {}

func (k *Kayak) Run(ctx context.Context) {}
