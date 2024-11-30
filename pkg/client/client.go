package client

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/tomquartz/pyxis-k8s/pkg/gateway"
	"github.com/tomquartz/pyxis-k8s/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Client struct {
	maxout      int
	profiles    []workload.TaskProfile
	ratioCumsum []float64
	sendChan    chan<- *workload.ClientRequest
	recvChan    <-chan *workload.ClientResponse
	results     []*workload.ClientResponse
	duration    time.Duration
}

func NewClient(maxout int, profiles []workload.TaskProfile) *Client {
	ratioCumsum := make([]float64, len(profiles))
	sum := 0.
	for i, profile := range profiles {
		if i != profile.TypeID {
			panic("profile typeID must be consecutive")
		}
		sum += profile.Percentage
		ratioCumsum[i] = sum
	}
	return &Client{
		maxout:      maxout,
		profiles:    profiles,
		ratioCumsum: ratioCumsum,
	}
}

func (c *Client) Connect(gateway *gateway.Gateway) {
	c.sendChan = gateway.Input()
	c.recvChan = gateway.Output()
}

func (c *Client) Run(ctx context.Context) {
	logger := log.FromContext(ctx)
	id := 0
	send := func() {
		id++
		c.sendChan <- c.newRequest(id)
	}
	start := time.Now()
	for i := 0; i < c.maxout; i++ {
		send()
	}
	for {
		select {
		case resp := <-c.recvChan:
			c.results = append(c.results, resp)
			if resp.Status != workload.SUCCESS {
				logger.Error(fmt.Errorf(resp.Result), "client received error response", "code", resp.Status)
			}
			send()
		case <-ctx.Done():
			c.duration = time.Since(start)
			return
		}
	}
}

func (c *Client) newRequest(id int) *workload.ClientRequest {
	x := rand.Float64()
	typeID := sort.SearchFloat64s(c.ratioCumsum, x)
	if typeID == len(c.profiles) {
		panic("invalid typeID")
	}
	profile := c.profiles[typeID]
	storageKeys := make([]string, profile.NumKV)
	for i := 0; i < profile.NumKV; i++ {
		storageKeys[i] = fmt.Sprintf("%d", rand.Intn(profile.NumKV))
	}
	return &workload.ClientRequest{
		ID:          fmt.Sprintf("%d", id),
		TypeID:      typeID,
		StorageKeys: storageKeys,
		ComputeSecs: profile.ComputeSecs,
	}
}

func (c *Client) Summary() string {
	// tput
	tput := float64(len(c.results)) / c.duration.Seconds()
	// slowdown
	prewarm := int(float64(len(c.results)) * 0.4)
	results := c.results[prewarm:]
	slowdowns := make([]float64, len(results))
	slowdownSum := 0.
	for _, resp := range results {
		this := resp.Latency.Seconds() / resp.ComputeTimeSecs
		slowdowns = append(slowdowns, this)
		slowdownSum += this
	}
	// percentiles
	sort.Float64s(slowdowns)
	slowdownAvg := slowdownSum / float64(len(slowdowns))
	slowdownP50 := slowdowns[len(slowdowns)/2]
	slowdownP99 := slowdowns[len(slowdowns)*99/100]
	// msg
	tputMsg := fmt.Sprintf("Throughput: %.0f req/s\n", tput)
	slowdownMsg := fmt.Sprintf("Slowdown: avg=%.1f p50=%.1f p99=%.1f\n", slowdownAvg, slowdownP50, slowdownP99)
	return tputMsg + slowdownMsg
}
