#!/bin/bash

STORAGE_SERVICE_URL="http://localhost:30081"
MEMORY_USAGE_BYTES=$(curl -s $STORAGE_SERVICE_URL/memory-usage)

echo "Pyxis storage server memory usage: $MEMORY_USAGE_BYTES bytes"