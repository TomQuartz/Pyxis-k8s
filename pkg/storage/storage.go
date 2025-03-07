package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	StorageServerChanSize        = 256
	StorageKVEntryAlignSizeBytes = 64
	KVAccessTimeSimulated        = 10 * time.Microsecond
)

type StorageServer struct {
	logger     logr.Logger
	workerChan chan interface{}
	nWorkers   int
	mu         sync.Mutex
	kv         map[string]string
}

func NewStorageServer(nWorkers int) *StorageServer {
	return &StorageServer{
		workerChan: make(chan interface{}, StorageServerChanSize),
		nWorkers:   nWorkers,
		kv:         make(map[string]string),
	}
}

func (s *StorageServer) ServeKV(w http.ResponseWriter, r *http.Request) {
	kvReq := &workload.StorageRequest{}
	err := json.NewDecoder(r.Body).Decode(kvReq)
	kvReq.SetResponseWriter(w)
	if err != nil {
		kvReq.Error(fmt.Errorf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	workerResps := make([]*workload.StorageResponse, len(kvReq.Keys))
	wg := sync.WaitGroup{}
	wg.Add(len(kvReq.Keys))
	for i := range kvReq.Keys {
		go func(i int) {
			defer wg.Done()
			req := &workload.StorageRequest{ID: fmt.Sprintf("%s-%d", kvReq.ID, i), Keys: []string{kvReq.Keys[i]}}
			if len(kvReq.Values) > i {
				req.Values = []string{kvReq.Values[i]}
			}
			req.SetResponseWriter(nil)
			s.workerChan <- req
			workerResps[i] = <-req.Done()
		}(i)
	}
	wg.Wait()
	kvResp := &workload.StorageResponse{ID: kvReq.ID}
	for _, r := range workerResps {
		if r.Error != nil {
			kvReq.Error(r.Error, http.StatusInternalServerError)
			break
		}
		kvResp.Keys = append(kvResp.Keys, r.Keys...)
		kvResp.Values = append(kvResp.Values, r.Values...)
	}
	if err := kvReq.Reply(kvResp); err != nil {
		s.logger.Error(err, "server failed to reply", "request", kvReq.ID)
	}
}

func (s *StorageServer) ServePushdown(w http.ResponseWriter, r *http.Request) {
	req := &workload.ClientRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	req = req.SetResponseWriter(w)
	if err != nil {
		req.Error(fmt.Errorf("server failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	s.workerChan <- req
	<-req.Done()
}

func (s *StorageServer) ServeMemoryUsageQuery(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roundUpMemoryUsage := func(u int) int {
		return (u + StorageKVEntryAlignSizeBytes - 1) / StorageKVEntryAlignSizeBytes * StorageKVEntryAlignSizeBytes
	}
	total := 0
	for k, v := range s.kv {
		total += roundUpMemoryUsage(len(k))
		total += roundUpMemoryUsage(len(v))
	}
	fmt.Fprintf(w, "%d\n", total)
}

func (s *StorageServer) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	s.logger = logger
	for i := 0; i < s.nWorkers; i++ {
		w := NewStorageWorker(i, s)
		go w.Run(ctx)
	}
	defer close(s.workerChan)

	logger.Info("Starting storage server", "nWorkers", s.nWorkers)
	http.HandleFunc(workload.StorageKVPath, s.ServeKV)
	http.HandleFunc(workload.StoragePushdownPath, s.ServePushdown)
	http.HandleFunc(workload.StorageMemoryUsageMetricPath, s.ServeMemoryUsageQuery)
	if err := http.ListenAndServe(workload.StorageListenPort, nil); err != http.ErrServerClosed {
		logger.Error(err, "Failed to run storage server")
	} else {
		logger.Info("Storage server stopped")
	}
}

type StorageWorker struct {
	id     int
	input  chan interface{}
	server *StorageServer
}

func NewStorageWorker(id int, server *StorageServer) *StorageWorker {
	return &StorageWorker{id: id, input: server.workerChan, server: server}
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
	defer req.Close()
	if len(req.Keys) != 1 {
		panic("internal error: multiple keys sent to worker")
	}
	logger.V(1).Info("worker processing kv request", "id", req.ID, "key", req.Keys[0])
	resp := &workload.StorageResponse{ID: req.ID}
	time.Sleep(KVAccessTimeSimulated)
	if len(req.Values) > 0 {
		w.put(req.Keys[0], req.Values[0])
	} else {
		key := req.Keys[0]
		value, ok := w.get(key)
		if !ok {
			value = "N/A"
		}
		resp.Keys = []string{key}
		resp.Values = []string{value}
	}
	if err := req.Reply(resp); err != nil {
		logger.Error(err, "worker failed to reply", "request", req.ID)
	}
	logger.V(1).Info("worker finished kv request", "request", req.ID)
}

func (w *StorageWorker) HandlePushdown(logger logr.Logger, req *workload.ClientRequest) {
	if req.ResponseWriter == nil {
		panic("missing response writer")
	}
	defer req.Close()
	logger.V(1).Info("worker processing pushdown request", "request", req.ID)

	if req.DefaultFuncRequest != nil {
		w.HandleDefaultFunc(logger, req)
	} else if req.PointerChasingFuncRequest != nil {
		w.HandlePointerChasing(logger, req)
	} else {
		req.Error(fmt.Errorf("unknown request"), http.StatusBadRequest)
	}
}

func (w *StorageWorker) HandleDefaultFunc(logger logr.Logger, req *workload.ClientRequest) {
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
	logger.V(1).Info("worker finished pushdown default func", "request", req.ID)
}

func (w *StorageWorker) HandlePointerChasing(logger logr.Logger, req *workload.ClientRequest) {
	// kv accesses
	kvStartTime := time.Now()
	key := req.PointerChasingFuncRequest.InitialKey
	for i := 0; i < req.PointerChasingFuncRequest.NumHops; i++ {
		logger.Info(fmt.Sprintf("Hop #%d: accessing %s", i, key))
		time.Sleep(KVAccessTimeSimulated)
		if next, ok := w.get(key); ok {
			key = next
		} else {
			break
		}
	}
	kvTime := time.Since(kvStartTime)
	logger.Info(fmt.Sprintf("Pointer chasing stopped at %s", key))

	// reply
	resp := &workload.ClientResponse{
		ID:              req.ID,
		Result:          key,
		StorageTimeSecs: kvTime.Seconds(),
	}
	if err := req.Reply(resp); err != nil {
		logger.Error(err, "worker failed to reply", "request", req.ID)
	}
	logger.V(1).Info("worker finished pushdown pointer chasing func", "request", req.ID)
}

func (w *StorageWorker) put(key, value string) {
	w.server.mu.Lock()
	defer w.server.mu.Unlock()
	w.server.kv[key] = value
}

func (w *StorageWorker) get(key string) (string, bool) {
	w.server.mu.Lock()
	defer w.server.mu.Unlock()
	value, ok := w.server.kv[key]
	return value, ok
}
