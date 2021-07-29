#!/bin/bash

go test -v -mod=vendor -cover -timeout=3600s -coverprofile=cover.out -coverpkg=github.com/polarismesh/polaris-server/naming,github.com/polarismesh/polaris-server/naming/batch,github.com/polarismesh/polaris-server/naming/cache,github.com/polarismesh/polaris-server/naming/auth,github.com/polarismesh/polaris-server/store/defaultStore,github.com/polarismesh/polaris-server/plugin/ratelimit/tokenBucket,github.com/polarismesh/polaris-server/common/model | tee test.log
go tool cover -html=cover.out -o index.html
coverage=$(cat test.log | grep "coverage:" | awk '{print $2}' | cut -d '%' -f 1)
