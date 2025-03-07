package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const ComputeServerChanSize = 64

type ComputeServer struct {
	workerChan chan *workload.ClientRequest
	nWorkers   int
}

func NewComputeServer(nWorkers int) *ComputeServer {
	return &ComputeServer{
		workerChan: make(chan *workload.ClientRequest, ComputeServerChanSize),
		nWorkers:   nWorkers,
	}
}

func (s *ComputeServer) Serve(w http.ResponseWriter, r *http.Request) {
	req := &workload.ClientRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	req = req.SetResponseWriter(w)
	if err != nil {
		req.Error(fmt.Errorf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}
	s.workerChan <- req
	<-req.Done()
}

func (s *ComputeServer) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	for i := 0; i < s.nWorkers; i++ {
		w := NewComputeWorker(i, s.workerChan)
		go w.Run(ctx)
	}
	defer close(s.workerChan)

	logger.Info("Starting compute server", "nWorkers", s.nWorkers)
	http.HandleFunc("/", s.Serve)
	if err := http.ListenAndServe(workload.ComputeListenPort, nil); err != http.ErrServerClosed {
		logger.Error(err, "Failed to run compute server")
	} else {
		logger.Info("Compute server stopped")
	}
}

type ComputeWorker struct {
	id    int
	input chan *workload.ClientRequest
}

func NewComputeWorker(id int, input chan *workload.ClientRequest) *ComputeWorker {
	return &ComputeWorker{id: id, input: input}
}

func (w *ComputeWorker) Run(ctx context.Context) {
	logger := log.FromContext(ctx).WithValues("id", w.id)
	for req := range w.input {
		w.HandleRequest(logger, req)
	}
}

func (w *ComputeWorker) HandleRequest(logger logr.Logger, req *workload.ClientRequest) {
	if req.ResponseWriter == nil {
		panic("missing response writer")
	}
	defer req.Close()
	logger.V(1).Info("processing request", "request", req.ID)
	if req.DefaultFuncRequest != nil {
		w.HandleDefaultFunc(logger, req)
	} else if req.PointerChasingFuncRequest != nil {
		w.HandlePointerChasing(logger, req)
	} else {
		req.Error(fmt.Errorf("unknown request"), http.StatusBadRequest)
	}
}

func (w *ComputeWorker) HandleDefaultFunc(logger logr.Logger, req *workload.ClientRequest) {
	// kv accesses
	kvStartTime := time.Now()
	kvReq := &workload.StorageRequest{
		ID:   fmt.Sprintf("%s-kv", req.ID),
		Keys: req.StorageKeys,
	}
	kvReqJson, err := json.Marshal(kvReq)
	if err != nil {
		req.Error(fmt.Errorf("failed to encode kv req: %v", err), http.StatusBadRequest)
		return
	}
	kvResp, err := http.Post(workload.StorageKVInternalURL, "application/json", bytes.NewReader(kvReqJson))
	if err != nil {
		req.Error(fmt.Errorf("failed to post kv req: %v", err), http.StatusInternalServerError)
		return
	}
	defer kvResp.Body.Close()
	kvTime := time.Since(kvStartTime)

	// compute
	computeStartTime := time.Now()
	time.Sleep(time.Duration(req.ComputeSecs * float64(time.Second)))
	computeTime := time.Since(computeStartTime)

	// reply
	logger.V(1).Info("finish request", "request", req.ID)
	resp := &workload.ClientResponse{
		ID:              req.ID,
		StorageTimeSecs: kvTime.Seconds(),
		ComputeTimeSecs: computeTime.Seconds(),
	}
	if err := req.Reply(resp); err != nil {
		logger.Error(err, "failed to reply", "request", req.ID)
	}
	logger.V(1).Info("Finish default func", "request", req.ID)
}

func (w *ComputeWorker) HandlePointerChasing(logger logr.Logger, req *workload.ClientRequest) {
	// kv accesses
	kvStartTime := time.Now()
	key := req.PointerChasingFuncRequest.InitialKey
	for i := 0; i < req.PointerChasingFuncRequest.NumHops; i++ {
		kvReq := &workload.StorageRequest{
			ID:   fmt.Sprintf("%s-%d", req.ID, i),
			Keys: []string{key},
		}
		kvReqJson, err := json.Marshal(kvReq)
		if err != nil {
			req.Error(fmt.Errorf("failed to encode kv req: %v", err), http.StatusBadRequest)
			return
		}
		httpResp, err := http.Post(workload.StorageKVInternalURL, "application/json", bytes.NewReader(kvReqJson))
		if err != nil {
			req.Error(fmt.Errorf("failed to post kv req: %v", err), http.StatusInternalServerError)
			return
		}
		defer httpResp.Body.Close()
		resp := &workload.StorageResponse{}
		if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
			req.Error(fmt.Errorf("failed to decode kv resp: %v", err), http.StatusInternalServerError)
			return
		}
		key = resp.Values[0]
		if key == "N/A" {
			break
		}
	}
	kvTime := time.Since(kvStartTime)

	// reply
	logger.V(1).Info("finish request", "request", req.ID)
	resp := &workload.ClientResponse{
		ID:              req.ID,
		Result:          key,
		StorageTimeSecs: kvTime.Seconds(),
	}
	if err := req.Reply(resp); err != nil {
		logger.Error(err, "failed to reply", "request", req.ID)
	}
	logger.V(1).Info("Finish pointer chasing func", "request", req.ID)
}
