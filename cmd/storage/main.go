package main

import (
	"flag"

	"github.com/tomquartz/pyxis-k8s/pkg/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var nWorkers int
var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "Enable debug log")
	flag.IntVar(&nWorkers, "workers", 8, "Number of workers to run in the storage server")
	flag.Parse()

	opts := ctrlzap.Options{
		Development: true,
	}
	if !debug {
		opts.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	ctrl.SetLogger(ctrlzap.New(ctrlzap.UseFlagOptions(&opts)))

	storageServer := storage.NewStorageServer(nWorkers)
	storageServer.Run(ctrl.SetupSignalHandler())
}
