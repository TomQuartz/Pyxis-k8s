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
	workerChan chan interface{}
	nWorkers   int
}

func NewStorageServer(nWorkers int) *StorageServer {
	return &StorageServer{
		workerChan: make(chan interface{}, StorageServerChanSize),
		nWorkers:   nWorkers,
	}
}

func (s *StorageServer) ServeKV(w http.ResponseWriter, r *http.Request) {
	kvReq := &workload.StorageRequest{}
	err := json.NewDecoder(r.Body).Decode(kvReq)
	if err != nil {
		kvReq.SetResponseWriter(w).Error(fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	for i, key := range kvReq.Keys {
		req := &workload.StorageRequest{ID: fmt.Sprintf("%s-%d", kvReq.ID, i), Keys: []string{key}}
		req = req.SetResponseWriter(w)
		s.workerChan <- req
		<-req.Done()
	}
}

func (s *StorageServer) ServePushdown(w http.ResponseWriter, r *http.Request) {
	req := &workload.ClientRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	req = req.SetResponseWriter(w)
	if err != nil {
		req.Error(fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	s.workerChan <- req
	<-req.Done()
}

func (s *StorageServer) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	for i := 0; i < s.nWorkers; i++ {
		w := NewStorageWorker(i, s.workerChan)
		go w.Run(ctx)
	}
	defer close(s.workerChan)

	logger.Info("Starting storage server", "nWorkers", s.nWorkers)
	http.HandleFunc(workload.StorageKVPath, s.ServeKV)
	http.HandleFunc(workload.StoragePushdownPath, s.ServePushdown)
	if err := http.ListenAndServe(workload.StorageListenPort, nil); err != http.ErrServerClosed {
		logger.Error(err, "Failed to run storage server")
	} else {
		logger.Info("Storage server stopped")
	}
}

type StorageWorker struct {
	id    int
	input chan interface{}
}

func NewStorageWorker(id int, input chan interface{}) *StorageWorker {
	return &StorageWorker{id: id, input: input}
}

func (w *StorageWorker) Run(ctx context.Context) {
	logger := log.FromContext(ctx).WithValues("id", w.id)
	for req := range w.input {
		w.HandleRequest(logger, req)
	}
}

func (w *StorageWorker) HandleRequest(logger logr.Logger, req interface{}) {
	switch req := req.(type) {
	case *workload.StorageRequest:
		w.HandleKV(logger, req)
	case *workload.ClientRequest:
		w.HandlePushdown(logger, req)
	default:
		logger.Error(fmt.Errorf("unknown request type: %T", req), "failed to handle request")
	}
}

func (w *StorageWorker) HandleKV(logger logr.Logger, req *workload.StorageRequest) {
	if req.ResponseWriter == nil {
		panic("missing response writer")
	}
	defer req.Close()
	logger.V(1).Info("processing kv request", "request", req.ID)
	time.Sleep(KVAccessTimeSimulated)
	if err := req.Reply(&workload.StorageResponse{ID: req.ID, Value: "value"}); err != nil {
		logger.Error(err, "failed to reply", "request", req.ID)
	}
	logger.V(1).Info("finish kv request", "request", req.ID)
}

func (w *StorageWorker) HandlePushdown(logger logr.Logger, req *workload.ClientRequest) {
	if req.ResponseWriter == nil {
		panic("missing response writer")
	}
	defer req.Close()
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
		ID:              req.ID,
		StorageTimeSecs: kvTime.Seconds(),
		ComputeTimeSecs: computeTime.Seconds(),
	}
	if err := req.Reply(resp); err != nil {
		logger.Error(err, "failed to reply", "request", req.ID)
	}
	logger.V(1).Info("finish pushdown request", "request", req.ID)
}
