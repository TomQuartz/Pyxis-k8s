package workload

import (
	"encoding/json"
	"net/http"
	"time"
)

type TaskProfile struct {
	TypeID      int     `json:"typeID"`
	Percentage  float64 `json:"percentage"`
	NumKV       int     `json:"numKV"`
	ComputeSecs float64 `json:"computeSecs"`
}

type DefaultFuncRequest struct {
	StorageKeys []string `json:"storageKeys"`
	ComputeSecs float64  `json:"computeSecs"`
}

type PointerChasingFuncRequest struct {
	InitialKey string `json:"initialKey"`
	NumHops    int    `json:"numHops"`
}

type ClientRequest struct {
	ID     string `json:"id"`
	TypeID int    `json:"typeID"`
	*DefaultFuncRequest
	*PointerChasingFuncRequest
	ResponseWriter http.ResponseWriter
	done           chan struct{}
}

func (c *ClientRequest) SetResponseWriter(w http.ResponseWriter) *ClientRequest {
	c.ResponseWriter = w
	c.done = make(chan struct{})
	return c
}

func (c *ClientRequest) Done() <-chan struct{} {
	return c.done
}

func (c *ClientRequest) Close() {
	close(c.done)
}

func (c *ClientRequest) Error(msg string, code int) {
	http.Error(c.ResponseWriter, msg, code)
}

func (c *ClientRequest) Reply(response *ClientResponse) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(http.StatusOK)
	return json.NewEncoder(c.ResponseWriter).Encode(response)
}

const (
	SUCCESS = iota
	FAIL_MARSHAL
	FAIL_SCHEDULE
	FAIL_SEND
	FAIL_EXECUTE
	FAIL_UNMARSHAL
)

type ClientResponse struct {
	ID              string  `json:"id"`
	Status          int     `json:"status,omitempty"`
	Result          string  `json:"result,omitempty"`
	StorageTimeSecs float64 `json:"storageTimeSecs"`
	ComputeTimeSecs float64 `json:"computeTimeSecs"`
	Latency         time.Duration
}

type StorageRequest struct {
	ID             string   `json:"id"`
	Keys           []string `json:"keys,omitempty"`
	Values         []string `json:"values,omitempty"`
	ResponseWriter http.ResponseWriter
	done           chan struct{}
}

type StorageResponse struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

func (s *StorageRequest) SetResponseWriter(w http.ResponseWriter) *StorageRequest {
	s.ResponseWriter = w
	s.done = make(chan struct{})
	return s
}

func (s *StorageRequest) Done() <-chan struct{} {
	return s.done
}

func (s *StorageRequest) Close() {
	close(s.done)
}

func (s *StorageRequest) Error(msg string, code int) {
	http.Error(s.ResponseWriter, msg, code)
}

func (s *StorageRequest) Reply(response *StorageResponse) error {
	s.ResponseWriter.Header().Set("Content-Type", "application/json")
	s.ResponseWriter.WriteHeader(http.StatusOK)
	return json.NewEncoder(s.ResponseWriter).Encode(response)
}
