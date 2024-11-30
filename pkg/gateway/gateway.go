package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tomquartz/pyxis-k8s/pkg/gateway/arbiter"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
)

type Gateway struct {
	requestChan  chan *workload.ClientRequest
	responseChan chan *workload.ClientResponse
	arbiter      arbiter.Arbiter
}

func NewGateway(maxout int, arbiter arbiter.Arbiter) *Gateway {
	return &Gateway{
		requestChan:  make(chan *workload.ClientRequest, maxout),
		responseChan: make(chan *workload.ClientResponse, maxout),
		arbiter:      arbiter,
	}
}

func (g *Gateway) Input() chan<- *workload.ClientRequest {
	return g.requestChan
}

func (g *Gateway) Output() <-chan *workload.ClientResponse {
	return g.responseChan
}

func (g *Gateway) Run(ctx context.Context) {
	go g.arbiter.Run(ctx)
	for {
		select {
		case req := <-g.requestChan:
			go g.handleRequest(ctx, req)
		case <-ctx.Done():
			return
		}
	}
}

// assume req is assigned ID
func (g *Gateway) handleRequest(ctx context.Context, req *workload.ClientRequest) {
	resp := &workload.ClientResponse{}
	defer func() {
		resp.ID = req.ID
		g.responseChan <- resp
	}()
	reqBytes, err := json.Marshal(req)
	if err != nil {
		resp.Status = workload.FAIL_MARSHAL
		resp.Result = err.Error()
		return
	}
	start := time.Now()
	// schedule
	decision := g.arbiter.Schedule(req)
	var postURL string
	switch decision {
	case arbiter.ToCompute:
		postURL = workload.ComputeServiceURL
	case arbiter.ToStorage:
		postURL = workload.StoragePushdownServiceURL
	default:
		resp.Status = workload.FAIL_SCHEDULE
		resp.Result = "invalid arbiter decision"
		return
	}
	// post
	postCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(postCtx, http.MethodPost, postURL, bytes.NewReader(reqBytes))
	if err != nil {
		resp.Status = workload.FAIL_SEND
		resp.Result = err.Error()
		return
	}
	// httpResp, err := http.Post(postURL, "application/json", bytes.NewReader(reqBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		resp.Status = workload.FAIL_SEND
		resp.Result = err.Error()
		return
	}
	defer httpResp.Body.Close()
	// decode
	if httpResp.StatusCode != http.StatusOK {
		resp.Status = workload.FAIL_EXECUTE
		if msg, err := io.ReadAll(httpResp.Body); err != nil {
			resp.Result = fmt.Sprintf("request failed with status: %v | failed to read response body: %v", resp.Status, err)
		} else {
			resp.Result = fmt.Sprintf("request failed with status: %v | %v", resp.Status, string(msg))
		}
		return
	}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		resp.Status = workload.FAIL_UNMARSHAL
		resp.Result = err.Error()
		return
	}
	resp.Status = workload.SUCCESS
	resp.Latency = time.Since(start)
	if resp.ComputeTimeSecs <= 0 || resp.StorageTimeSecs <= 0 {
		resp.Status = workload.FAIL_UNMARSHAL
		resp.Result = "invalid response: zero compute or storage time: " + resp.Result
	}
}
