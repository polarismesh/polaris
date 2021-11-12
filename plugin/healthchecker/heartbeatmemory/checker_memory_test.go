package heartbeatmemory

import (
	"github.com/polarismesh/polaris-server/plugin"
	"sync"
	"testing"
)

func TestMemoryHealthChecker_Query(t *testing.T) {
	mhc := MemoryHealthChecker{
		hbRecords: new(sync.Map),
	}
	test := HeartbeatRecord{
		Server:     "127.0.0.1",
		CurTimeSec: 1,
	}
	mhc.hbRecords.Store("key", test)

	queryRequest := plugin.QueryRequest{
		InstanceId: "key",
		Host:       "127.0.0.2",
		Port:       80,
		Healthy:    true,
	}
	qr, err := mhc.Query(&queryRequest)
	if err != nil {
		t.Error(err)
	}
	if qr.Server != "127.0.0.1" {
		t.Error()
	}
	if qr.LastHeartbeatSec != 1 {
		t.Error()
	}

}
