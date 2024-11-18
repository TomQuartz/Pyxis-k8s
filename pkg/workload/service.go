package workload

const (
	// compute
	ComputeServiceName = "pyxis-compute" + ".svc.cluster.local"
	ComputeServicePort = ":8080"
	ComputeURL         = "http://" + ComputeServiceName + ComputeServicePort
	// storage
	StorageServiceName  = "pyxis-storage" + ".svc.cluster.local"
	StorageServicePort  = ":8081"
	StorageKVPath       = "/kv"
	StoragePushdownPath = "/pushdown"
	StorageKVURL        = "http://" + StorageServiceName + StorageServicePort + StorageKVPath
	StoragePushdownURL  = "http://" + StorageServiceName + StorageServicePort + StoragePushdownPath
)
