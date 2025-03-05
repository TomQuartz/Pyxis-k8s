package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tomquartz/pyxis-k8s/demo/pkg/workload"
)

func doStorageSideExecution(req *workload.ClientRequest) (*workload.ClientResponse, error) {
	reqJson, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode req: %v", err)
	}
	httpResp, err := http.Post(workload.StoragePushdownServiceURL, "application/json", bytes.NewReader(reqJson))
	if err != nil {
		return nil, fmt.Errorf("failed to post req: %v", err)
	}
	defer httpResp.Body.Close()
	resp := &workload.ClientResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("failed to decode resp: %v", err)
	}
	return resp, nil
}

func doComputeSideExecution(req *workload.ClientRequest) (*workload.ClientResponse, error) {
	// kv accesses
	kvStartTime := time.Now()
	key := req.PointerChasingFuncRequest.InitialKey
	for i := 0; i < req.PointerChasingFuncRequest.NumHops; i++ {
		fmt.Printf("Hop #%d: accessing %s\n", i, key)
		kvReq := &workload.StorageRequest{
			ID:   fmt.Sprintf("%s-%d", req.ID, i),
			Keys: []string{key},
		}
		kvReqJson, err := json.Marshal(kvReq)
		if err != nil {
			return nil, fmt.Errorf("failed to encode kv req: %v", err)
		}
		httpResp, err := http.Post(workload.StorgeKVServiceURL, "application/json", bytes.NewReader(kvReqJson))
		if err != nil {
			return nil, fmt.Errorf("failed to post kv req: %v", err)
		}
		defer httpResp.Body.Close()
		resp := &workload.StorageResponse{}
		if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
			return nil, fmt.Errorf("failed to decode kv resp: %v", err)
		}
		key = resp.Value
		if key == "N/A" {
			break
		}
	}
	kvTime := time.Since(kvStartTime)
	fmt.Printf("Pointer chasing stopped at %s", key)

	resp := &workload.ClientResponse{
		ID:              req.ID,
		Result:          key,
		StorageTimeSecs: kvTime.Seconds(),
	}
	return resp, nil
}

func doKVReq(req *workload.StorageRequest) (string, error) {
	kvReqJson, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to encode kv req: %v", err)
	}
	httpResp, err := http.Post(workload.StorgeKVServiceURL, "application/json", bytes.NewReader(kvReqJson))
	if err != nil {
		return "", fmt.Errorf("failed to post kv req: %v", err)
	}
	defer httpResp.Body.Close()
	resp := &workload.StorageResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return "", fmt.Errorf("failed to decode kv resp: %v", err)
	}
	return resp.Value, nil
}

func populateStorageServer(nKeys int) {
	keys := make([]string, nKeys)
	values := make([]string, nKeys)
	for i := 0; i < nKeys; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	for i := 0; i < nKeys; i++ {
		values[i] = keys[(i+1)%nKeys]
	}
	req := &workload.StorageRequest{
		ID:     "populate",
		Keys:   keys,
		Values: values,
	}
	if _, err := doKVReq(req); err != nil {
		panic(fmt.Sprintf("failed to populate storage server: %v", err))
	}
}
