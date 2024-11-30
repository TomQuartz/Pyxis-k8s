package workload

const (
	// compute
	ComputeListenPort      = ":8080"
	ComputeServiceNodePort = ":30080"
	ComputeServiceURL      = "http://localhost" + ComputeServiceNodePort
	// storage
	StorageListenPort      = ":8081"
	StorageServiceNodePort = ":30081"
	// compute-to-storage (in-cluster)
	StorageKVPath        = "/kv"
	StorageKVInternalURL = "http://pyxis-storage" + StorageKVPath // resolves to pyxis-storage.<ns>.svc.cluster.local
	// client-to-storage (out-of-cluster)
	StoragePushdownPath       = "/pushdown"
	StoragePushdownServiceURL = "http://localhost" + StorageServiceNodePort + StoragePushdownPath
)
