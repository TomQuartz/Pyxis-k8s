package workload

const (
	// compute
	ComputeListenPort      = ":8080"
	ComputeServiceNodePort = ":30080"
	ComputeServiceURL      = "http://localhost" + ComputeServiceNodePort
	// storage
	StorageListenPort      = ":8081"
	StorageServiceNodePort = ":30081"
	// kv service
	StorageKVPath = "/kv"
	// compute-to-storage (in-cluster)
	StorageKVInternalURL = "http://pyxis-storage" + StorageKVPath // resolves to pyxis-storage.<ns>.svc.cluster.local
	// client-to-storage (out-of-cluster)
	StorgeKVServiceURL = "http://localhost" + StorageServiceNodePort + StorageKVPath
	// pushdown service
	StoragePushdownPath = "/pushdown"
	// client-to-storage (out-of-cluster)
	StoragePushdownServiceURL = "http://localhost" + StorageServiceNodePort + StoragePushdownPath
	// memory usage metric service
	StorageMemoryUsageMetricPath = "/memory-usage"
	// client-to-storage (out-of-cluster)
	StorageMemoryUsageMetricServiceURL = "http://localhost" + StorageServiceNodePort + StorageMemoryUsageMetricPath
)
