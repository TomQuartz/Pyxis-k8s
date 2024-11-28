package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tomquartz/pyxis-k8s/pkg/client"
	"github.com/tomquartz/pyxis-k8s/pkg/gateway"
	"github.com/tomquartz/pyxis-k8s/pkg/gateway/arbiter"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var nSeconds int
var maxout int
var configDir string
var arbiterFramework string
var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "Enable debug log")
	flag.IntVar(&nSeconds, "time", 60, "Number of seconds to run the client")
	flag.IntVar(&maxout, "maxout", 8, "Number of requests to send concurrently")
	flag.StringVar(&arbiterFramework, "arbiter", "pyxis", "Arbiter framework. Options: kayak, pyxis")
	flag.StringVar(&configDir, "config", "manifests", "Path to json config file directory")
	flag.Parse()

	opts := ctrlzap.Options{
		Development: true,
	}
	if !debug {
		opts.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	ctrl.SetLogger(ctrlzap.New(ctrlzap.UseFlagOptions(&opts)))

	// find config dir
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		ctrl.Log.Error(err, "Failed to find config dir")
		return
	}
	if info, err := os.Stat(configDir); err != nil || !info.IsDir() {
		ctrl.Log.Error(err, "Invalid config dir")
		return
	}

	// read task profiles
	var profiles []workload.TaskProfile
	tasksBytes, err := os.ReadFile(filepath.Join(configDir, "tasks.json"))
	if err != nil {
		ctrl.Log.Error(err, "Failed to read task profiles from tasks.json")
		return
	}
	if err := json.Unmarshal(tasksBytes, &profiles); err != nil {
		ctrl.Log.Error(err, "Failed to unmarshal task profiles")
		return
	}

	// read arbiter config
	arbiterBytes, err := os.ReadFile(filepath.Join(configDir, arbiterFramework+".json"))
	if err != nil {
		ctrl.Log.Error(err, "Failed to read arbiter config from "+arbiterFramework+".json")
		return
	}

	// create arbiter
	var arbiterImpl arbiter.Arbiter
	switch arbiterFramework {
	case "kayak":
		var arbiterConfig arbiter.KayakConfig
		if err := json.Unmarshal(arbiterBytes, &arbiterConfig); err != nil {
			ctrl.Log.Error(err, "Failed to unmarshal arbiter config from "+arbiterFramework+".json")
			return
		}
		arbiterConfig.TaskProfiles = profiles
		arbiterImpl = arbiter.NewKayak(&arbiterConfig)
	case "pyxis":
		var arbiterConfig arbiter.PyxisConfig
		if err := json.Unmarshal(arbiterBytes, &arbiterConfig); err != nil {
			ctrl.Log.Error(err, "Failed to unmarshal arbiter config from "+arbiterFramework+".json")
			return
		}
		arbiterConfig.TaskProfiles = profiles
		arbiterImpl = arbiter.NewPyxis(&arbiterConfig)
	}

	// create gateway
	gw := gateway.NewGateway(maxout, arbiterImpl)

	// create client
	cl := client.NewClient(maxout, profiles)
	cl.Connect(gw)

	// run
	ctrl.Log.Info(fmt.Sprintf("Running for %v seconds", nSeconds))
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Duration(nSeconds) * time.Second)
		cancel()
	}()
	go gw.Run(ctx)
	cl.Run(ctx)

	fmt.Println("Finished")
	fmt.Println(cl.Summary())
}
