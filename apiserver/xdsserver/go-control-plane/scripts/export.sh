#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

DIRS=(  "pkg/cache"
        "pkg/server"
        "pkg/test/resource"
        "pkg/test"
)
