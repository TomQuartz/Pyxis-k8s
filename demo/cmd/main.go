package main

import (
	"flag"
	"fmt"

	"github.com/tomquartz/pyxis-k8s/demo/pkg/workload"
)

var numHops int
var numKeys int

func main() {
	flag.IntVar(&numHops, "hops", 3, "Number of hops in pointer chasing")
	flag.IntVar(&numKeys, "keys", 1000, "Number of keys in the data store")
	flag.Parse()

	fmt.Printf("################### Pointer Chasing ###################n\n")
	fmt.Printf("Number of hops: %d\n", numHops)
	fmt.Printf("########################################################\n\n")

	populateStorageServer(numKeys)

	req := &workload.ClientRequest{
		PointerChasingFuncRequest: &workload.PointerChasingFuncRequest{
			InitialKey: "key-0",
			NumHops:    numHops,
		},
	}

	fmt.Printf("\n\n################### Compute-side Execution ###################\n\n")
	req.ID = "compute-side"
	_, cErr := doComputeSideExecution(req)
	if cErr != nil {
		fmt.Printf("Failed to execute on compute side: %v\n", cErr)
		return
	}
	fmt.Println("Finish compute-side pointer chasing func")

	fmt.Printf("\n\n################### Storage-side Execution ###################\n\n")
	req.ID = "storage-side"
	_, sErr := doStorageSideExecution(req)
	if sErr != nil {
		fmt.Printf("Failed to execute on storage side: %v\n", sErr)
		return
	}
	fmt.Println("Finish storage-side pointer chasing func")
}
