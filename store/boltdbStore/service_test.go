package boltdbStore

import (
	"fmt"
	"github.com/polarismesh/polaris-server/common/model"
	"strconv"
	"testing"
	"time"
)

const (
	serviceCount = 5
	aliasCount = 3
)

func TestServiceStore_AddService(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	for i := 0; i < serviceCount; i++ {
		err := sStore.AddService(&model.Service{
			ID: "svcid"+strconv.Itoa(i),
			Name: "svcname"+strconv.Itoa(i),
			Namespace: "testsvc",
			Business: "testbuss",
			Ports: "8080",
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Comment: "testcomment",
			Department: "testdepart",
			Token: "testtoken",
			Owner: "testowner",
			Revision: "testrevision"+strconv.Itoa(i),
			Reference: "",
			Valid: true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < aliasCount; i++ {
		err := sStore.AddService(&model.Service{
			ID: "aliasid" + strconv.Itoa(i),
			Name: "aliasname " + strconv.Itoa(i),
			Namespace: "testsvc",
			Business: "testbuss",
			Ports: "8080",
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Comment: "testcomment",
			Department: "testdepart",
			Token: "testtoken",
			Owner: "testowner",
			Revision: "testrevision"+strconv.Itoa(i),
			Reference: "svcid"+strconv.Itoa(i),
			Valid: true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestServiceStore_GetServices(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	total, ss, err := sStore.GetServices(nil, nil, nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != routeCount + aliasCount {
		t.Fatal(fmt.Sprintf("service total count not match, expect %d, got %d", routeCount + aliasCount, total))
	}
	if len(ss) != routeCount + aliasCount {
		t.Fatal(fmt.Sprintf("service count not match, expect %d, got %d", routeCount + aliasCount, len(ss)))
	}
	for _, s := range ss {
		fmt.Printf("get service alias %+v\n", s)
	}
}

func TestServiceStore_GetServicesBatch(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	sArg := make([]*model.Service, 2)
	for i := 0; i < 2; i++ {
		sArg[i] = &model.Service{
			Name: "svcname"+strconv.Itoa(i),
			Namespace: "testsvc",
		}
	}

	ss, err := sStore.GetServicesBatch(sArg)
	if err != nil {
		t.Fatal(err)
	}

	if len(ss) != 2 {
		t.Fatal(fmt.Sprintf("get service count error, except %d, got %d", 2, len(ss)))
	}
}

func TestServiceStore_GetServiceByID(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	ss, err := sStore.GetServiceByID("svcid1")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service by id: %+v\n", ss)
}

func TestServiceStore_UpdateService(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.UpdateService(&model.Service{
		ID: "svcid1",
		Name: "modifyName1",
		Namespace: "modifyNamespace1",
		Token: "modifyToken1",
		Owner: "modifyOwner1",
		Revision: "modifyRevision1",

	}, true)
	if err != nil {
		t.Fatal(err)
	}

	// check update result
	ss, err := sStore.getServiceByID("svcid1")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service %+v\n", ss)

	if ss.Name != "modifyName1" ||
		ss.Namespace != "modifyNamespace1" ||
		ss.Token != "modifyToken1" ||
		ss.Owner != "modifyOwner1" ||
		ss.Revision != "modifyRevision1" ||
		ss.Reference != "" {
		t.Fatal(fmt.Sprintf("update service error"))
	}
}

func TestServiceStore_UpdateServiceToken(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.UpdateServiceToken("svcid1", "ttttt1", "rrrrrr1")
	if err != nil {
		t.Fatal(err)
	}

	// check update result
	ss, err := sStore.getServiceByID("svcid1")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service %+v\n", ss)

	if ss.Name != "modifyName1" ||
		ss.Namespace != "modifyNamespace1" ||
		ss.Token != "ttttt1" ||
		ss.Owner != "modifyOwner1" ||
		ss.Revision != "rrrrrr1" ||
		ss.Reference != "" {
		t.Fatal(fmt.Sprintf("update service error"))
	}
}

func TestServiceStore_GetSourceServiceToken(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	ss, err := sStore.GetSourceServiceToken("modifyName1", "modifyNamespace1")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service token: %+v\n", ss)
}

func TestServiceStore_GetService(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	ss, err := sStore.GetService("modifyName1", "modifyNamespace1")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service by name and namespace: %+v\n", ss)
}

func TestServiceStore_GetServiceAliases(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	total, ss, err := sStore.GetServiceAliases(nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != aliasCount {
		t.Fatal(fmt.Sprintf("service total count not match, expect %d, got %d", aliasCount, total))
	}
	if len(ss) != aliasCount {
		t.Fatal(fmt.Sprintf("service count not match, expect %d, got %d", aliasCount, len(ss)))
	}

	for _, s := range ss {
		fmt.Printf("get service alias %+v\n", s)
	}
}

func TestServiceStore_GetServicesCount(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	count, err := sStore.GetServicesCount()
	if err != nil {
		t.Fatal(err)
	}

	if count != routeCount + aliasCount {
		t.Fatal(fmt.Sprintf("get service count error, except %d, got %d", routeCount + aliasCount, count))
	}
}


func TestServiceStore_GetMoreServices(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	ss, err := sStore.GetService("svcname3", "testsvc")
	if err != nil {
		t.Fatal(err)
	}

	sss, err := sStore.GetMoreServices(ss.ModifyTime, true, false, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(sss) != routeCount + aliasCount - 3 {
		t.Fatal(fmt.Sprintf("get service count error, except %d, got %d", routeCount + aliasCount - 3, len(sss)))
	}
}


func TestServiceStore_UpdateServiceAlias(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.UpdateServiceAlias(&model.Service{
		ID: "svcid2",
		Name: "modifyName2",
		Namespace: "modifyNamespace2",
		Token: "modifyToken2",
		Owner: "modifyOwner2",
		Revision: "modifyRevision2",
		Reference: "1",

	}, true)
	if err != nil {
		t.Fatal(err)
	}

	// check update result
	ss, err := sStore.getServiceByID("svcid2")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service %+v\n", ss)

	if ss.Name != "modifyName2" ||
		ss.Namespace != "modifyNamespace2" ||
		ss.Token != "modifyToken2" ||
		ss.Owner != "modifyOwner2" ||
		ss.Revision != "modifyRevision2" ||
		ss.Reference != "1" {
		t.Fatal(fmt.Sprintf("update service error"))
	}
}


func TestServiceStore_DeleteServiceAlias(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.DeleteServiceAlias("svcname0", "testsvc")
	if err != nil {
		t.Fatal(err)
	}

	// check delete res
	total, s, _ := sStore.GetServices(nil, nil, nil, 0, 20)
	if total != routeCount + aliasCount - 1 || len(s) != routeCount + aliasCount - 1 {
		t.Fatal(fmt.Sprintf("delete service not success"))
	}
}

func TestServiceStore_DeleteService(t *testing.T){
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	total, ss, err := sStore.GetServices(nil, nil, nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range ss {
		fmt.Printf("get service %+v\n", s)
		err := sStore.DeleteService(s.ID, "", "")
		if err != nil{
			t.Fatal(err)
		}
	}

	// check delete res
	total, s, _ := sStore.GetServices(nil, nil, nil, 0, 20)
	if total != 0 || len(s) != 0 {
		t.Fatal(fmt.Sprintf("delete service not success"))
	}
}



