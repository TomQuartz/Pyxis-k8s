package arbiter

import (
	"context"

	"github.com/tomquartz/pyxis-k8s/pkg/workload"
)

type Arbiter interface {
	Run(ctx context.Context)
	Schedule(req *workload.ClientRequest) int
	Finish(resp *workload.ClientResponse)
}

const (
	ToCompute = iota
	ToStorage
)

type ArbiterConfig struct {
	IntervalSecs float64                `json:"intervalSecs"`
	StartPoint   float64                `json:"startPoint"`
	TaskProfiles []workload.TaskProfile `json:"taskProfiles"`
}

type KayakConfig struct {
	ArbiterConfig
}

type PyxisConfig struct {
	ArbiterConfig
	StepSizeRel    float64 `json:"stepSizeRel"`
	StopPrecision  float64 `json:"stopPrecision"`
	ReferencePoint float64 `json:"referencePoint,omitempty"`
}
