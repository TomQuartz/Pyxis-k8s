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

type ClientRequest struct {
	ID             string   `json:"id"`
	TypeID         int      `json:"typeID"`
	StorageKeys    []string `json:"storageKeys"`
	ComputeSecs    float64  `json:"computeSecs"`
	ResponseWriter http.ResponseWriter
}

func (c *ClientRequest) SetResponseWriter(w http.ResponseWriter) *ClientRequest {
	c.ResponseWriter = w
	return c
}

func (c *ClientRequest) Error(msg string, code int) {
	http.Error(c.ResponseWriter, msg, code)
}

func (c *ClientRequest) Reply(response *ClientResponse) {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(c.ResponseWriter).Encode(response)
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
	Status          int     `json:"status"`
	Result          string  `json:"result"`
	StorageTimeSecs float64 `json:"storageTimeSecs"`
	ComputeTimeSecs float64 `json:"computeTimeSecs"`
	Latency         time.Duration
}

type StorageRequest struct {
	ID             string         `json:"id"`
	Pushdown       *ClientRequest `json:"pushdown,omitempty"`
	Key            string         `json:"key,omitempty"`
	ResponseWriter http.ResponseWriter
}

type StorageResponse struct {
	ID    string
	Value string
}

func (s *StorageRequest) SetResponseWriter(w http.ResponseWriter) *StorageRequest {
	s.ResponseWriter = w
	return s
}

func (s *StorageRequest) Error(msg string, code int) {
	http.Error(s.ResponseWriter, msg, code)
}

func (s *StorageRequest) Reply(response *StorageResponse) {
	s.ResponseWriter.Header().Set("Content-Type", "application/json")
	s.ResponseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(s.ResponseWriter).Encode(response)
}
