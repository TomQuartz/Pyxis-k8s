package main

import (
	"flag"

	"github.com/tomquartz/pyxis-k8s/pkg/compute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var nWorkers int
var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "Enable debug log")
	flag.IntVar(&nWorkers, "workers", 8, "Number of workers to run in the compute server")
	flag.Parse()

	opts := ctrlzap.Options{
		Development: true,
	}
	if !debug {
		opts.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	ctrl.SetLogger(ctrlzap.New(ctrlzap.UseFlagOptions(&opts)))

	computeServer := compute.NewComputeServer(nWorkers)
	computeServer.Run(ctrl.SetupSignalHandler())
}
