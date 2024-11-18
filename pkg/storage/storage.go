package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const StorageServerChanSize = 256
const KVAccessTimeSimulated = 10 * time.Microsecond

type StorageServer struct {
	workerChan chan *workload.StorageRequest
	nWorkers   int
}

func NewStorageServer(nWorkers int) *StorageServer {
	return &StorageServer{
		workerChan: make(chan *workload.StorageRequest, StorageServerChanSize),
		nWorkers:   nWorkers,
	}
}

func (s *StorageServer) Serve(w http.ResponseWriter, r *http.Request) {
	req := &workload.StorageRequest{}
	req = req.SetResponseWriter(w)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		req.Error(fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	s.workerChan <- req
}

func (s *StorageServer) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	for i := 0; i < s.nWorkers; i++ {
		w := NewStorageWorker(i, s.workerChan)
		go w.Run(ctx)
	}
	defer close(s.workerChan)

	logger.Info("Starting storage server", "nWorkers", s.nWorkers)
	port := workload.StorageServicePort
	http.HandleFunc(workload.StorageKVPath, s.Serve)
	http.HandleFunc(workload.StoragePushdownPath, s.Serve)
	if err := http.ListenAndServe(port, nil); err != http.ErrServerClosed {
		logger.Error(err, "Failed to run storage server")
	} else {
		logger.Info("Storage server stopped")
	}
}

type StorageWorker struct {
	id    int
	input chan *workload.StorageRequest
}

func NewStorageWorker(id int, input chan *workload.StorageRequest) *StorageWorker {
	return &StorageWorker{id: id, input: input}
}

func (w *StorageWorker) Run(ctx context.Context) {
	logger := log.FromContext(ctx).WithValues("id", w.id)
	for req := range w.input {
		w.HandleRequest(logger, req)
	}
}

func (w *StorageWorker) HandleRequest(logger logr.Logger, req *workload.StorageRequest) {
	if req.ResponseWriter == nil {
		panic("missing response writer")
	}

	if req.Pushdown != nil {
		w.HandlePushdown(logger, req.Pushdown)
	} else {
		w.HandleKV(logger, req)
	}
}

func (w *StorageWorker) HandleKV(logger logr.Logger, req *workload.StorageRequest) {
	logger.V(1).Info("processing kv request", "request", req.ID)
	time.Sleep(KVAccessTimeSimulated)
	req.Reply(&workload.StorageResponse{ID: req.ID, Value: "value"})
}

func (w *StorageWorker) HandlePushdown(logger logr.Logger, req *workload.ClientRequest) {
	logger.V(1).Info("processing pushdown request", "request", req.ID)

	// kv accesses
	kvStartTime := time.Now()
	for range req.StorageKeys {
		time.Sleep(KVAccessTimeSimulated)
	}
	kvTime := time.Since(kvStartTime)

	// compute
	computeStartTime := time.Now()
	time.Sleep(time.Duration(req.ComputeSecs * float64(time.Second)))
	computeTime := time.Since(computeStartTime)

	// reply
	resp := &workload.ClientResponse{
		StorageTimeSecs: kvTime.Seconds(),
		ComputeTimeSecs: computeTime.Seconds(),
	}
	req.Reply(resp)
}
