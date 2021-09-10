package boltdbStore

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"strconv"
	"testing"
	"time"
)

const (
	insCount = 5
)

func TestInstanceStore_AddInstance(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	insStore := &instanceStore{handler: handler}
	for i := 0; i < insCount; i++ {

		nowt := time.Now().Format("2006-01-02 15:04:05")

		err = insStore.AddInstance(&model.Instance{
			Proto: &api.Instance{
				Id: &wrappers.StringValue{Value: "insid"+strconv.Itoa(i)},
				Service: &wrappers.StringValue{Value: "svcid1"},
				Namespace: &wrappers.StringValue{Value: "testns"},
				Host: &wrappers.StringValue{Value: "1.1.1."+strconv.Itoa(i)},
				Port: &wrappers.UInt32Value{Value: uint32(i)},
				Protocol: &wrappers.StringValue{Value: "grpc"},
				Weight: &wrappers.UInt32Value{Value: uint32(i)},
				EnableHealthCheck: &wrappers.BoolValue{Value: true},
				Healthy: &wrappers.BoolValue{Value: true},
				Isolate: &wrappers.BoolValue{Value: true},
				Metadata: map[string]string{
					"insk1": "insv1",
					"insk2": "insv2",
				},
				Ctime: &wrappers.StringValue{Value: nowt},
				Mtime: &wrappers.StringValue{Value: nowt},
				Revision: &wrappers.StringValue{Value: "revision"+strconv.Itoa(i)},
				ServiceToken: &wrappers.StringValue{Value: "token"+strconv.Itoa(i)},
			},
			ServiceID: "svcid1",
			ServicePlatformID: "svcPlatId1",
			Valid: true,
			ModifyTime: time.Now(),

		})
		if nil != err {
			t.Fatal(err)
		}
	}
}

func TestInstanceStore_GetExpandInstances(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	insStore := &instanceStore{handler: handler}

	total, ii, err := insStore.GetExpandInstances(nil, nil, 0, 20)
	if nil != err {
		t.Fatal(err)
	}
	if total != routeCount {
		t.Fatal(fmt.Sprintf("routing total count not match, expect %d, got %d", routeCount, total))
	}
	if len(ii) != routeCount {
		t.Fatal(fmt.Sprintf("routing count not match, expect %d, got %d", routeCount, len(ii)))
	}
	for _, i := range ii {
		fmt.Printf("routing conf is %+v\n", i)
	}

}

func TestInstanceStore_GetInstancesMainByService(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	insStore := &instanceStore{handler: handler}

	ii, err := insStore.GetInstancesMainByService("svcid1", "1.1.1.1")
	if nil != err {
		t.Fatal(err)
	}

	for _, i := range ii {
		fmt.Printf("get instance %+v\n", i)
	}

}