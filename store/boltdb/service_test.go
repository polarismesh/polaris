/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package boltdb

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/stretchr/testify/assert"
)

const (
	serviceCount = 5
	aliasCount   = 3
)

func TestServiceStore_AddService(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	for i := 0; i < serviceCount; i++ {
		err := sStore.AddService(&model.Service{
			ID:        "svcid" + strconv.Itoa(i),
			Name:      "svcname" + strconv.Itoa(i),
			Namespace: "testsvc",
			Business:  "testbuss",
			Ports:     "8080",
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Comment:    "testcomment",
			Department: "testdepart",
			Token:      "testtoken",
			Owner:      "testowner",
			Revision:   "testrevision" + strconv.Itoa(i),
			Reference:  "",
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < aliasCount; i++ {
		err := sStore.AddService(&model.Service{
			ID:        "aliasid" + strconv.Itoa(i),
			Name:      "aliasname " + strconv.Itoa(i),
			Namespace: "testsvc",
			Business:  "testbuss",
			Ports:     "8080",
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Comment:    "testcomment",
			Department: "testdepart",
			Token:      "testtoken",
			Owner:      "testowner",
			Revision:   "testrevision" + strconv.Itoa(i),
			Reference:  "svcid" + strconv.Itoa(i),
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestServiceStore_GetServices(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	serviceMetas := map[string]string{
		"k1": "v1",
	}

	_, ss, err := sStore.GetServices(nil, serviceMetas, nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ss {
		fmt.Printf("get service origin %+v\n", s)
	}
}

func TestServiceStore_GetServicesBatch(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	sArg := make([]*model.Service, 2)
	for i := 0; i < 2; i++ {
		sArg[i] = &model.Service{
			Name:      "svcname" + strconv.Itoa(i),
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

func TestServiceStore_GetServiceByID(t *testing.T) {
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

func TestServiceStore_UpdateService(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.UpdateService(&model.Service{
		ID:        "svcid1",
		Name:      "svcname1",
		Namespace: "testsvc",
		Token:     "modifyToken1",
		Meta: map[string]string{
			"k111": "v1111",
		},
		Owner:      "modifyOwner1",
		Revision:   "modifyRevision1",
		Department: "modifyDepartment",
		Business:   "modifyBusiness",
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

	if ss.Department != "modifyDepartment" || ss.Business != "modifyBusiness" ||
		ss.Reference != "" {
		t.Fatal("update service error")
	}
}

func TestServiceStore_UpdateServiceToken(t *testing.T) {
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

	if ss.Token != "ttttt1" ||
		ss.Revision != "rrrrrr1" ||
		ss.Reference != "" {
		t.Fatal("update service error")
	}
}

func TestServiceStore_GetSourceServiceToken(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	ss, err := sStore.GetSourceServiceToken("svcname1", "testsvc")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("get service token: %+v\n", ss)
}

func TestServiceStore_GetService(t *testing.T) {
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

func TestServiceStore_GetServiceAliases(t *testing.T) {
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

func TestServiceStore_GetServicesCount(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	_, err = sStore.GetServicesCount()
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceStore_FuzzyGetService(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	for i := 0; i < serviceCount; i++ {
		idxStr := strconv.Itoa(i)
		err := sStore.AddService(&model.Service{
			ID:        "fuzzsvcid" + idxStr,
			Name:      "fuzzsvcname" + idxStr,
			Namespace: "testsvc",
			Business:  "fuzztestbuss",
			Ports:     "8080",
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Comment:    "fuzztestcomment",
			Department: "fuzztestdepart",
			Token:      "testtoken",
			Owner:      "testowner",
			Revision:   "testrevision" + idxStr,
			Reference:  "",
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for i := 0; i < serviceCount; i++ {
			idxStr := strconv.Itoa(i)
			sStore.DeleteService("fuzzsvcid"+idxStr, "fuzzsvcname"+idxStr, "testsvc")
		}
	}()
	serviceFilters := make(map[string]string)
	serviceFilters["name"] = "fuzzsvcname*"

	count, _, err := sStore.GetServices(serviceFilters, nil, nil, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != serviceCount {
		t.Fatal(fmt.Sprintf("fuzzy query error, expect %d, actual %d", serviceCount, count))
	}

	serviceFilters["name"] = "fuzzsvcname"
	count, _, err = sStore.GetServices(serviceFilters, nil, nil, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal(fmt.Sprintf("fuzzy query error, expect %d, actual %d", 0, count))
	}

	serviceFilters = make(map[string]string)
	serviceFilters["department"] = "fuzztest*"

	count, _, err = sStore.GetServices(serviceFilters, nil, nil, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != serviceCount {
		t.Fatal(fmt.Sprintf("fuzzy query error, expect %d, actual %d", serviceCount, count))
	}

	serviceFilters = make(map[string]string)
	serviceFilters["business"] = "fuzztest*"

	count, _, err = sStore.GetServices(serviceFilters, nil, nil, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != serviceCount {
		t.Fatal(fmt.Sprintf("fuzzy query error, expect %d, actual %d", serviceCount, count))
	}

}

func TestServiceStore_GetMoreServices(t *testing.T) {
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

	_, err = sStore.GetMoreServices(ss.ModifyTime, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceStore_UpdateServiceAlias(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	err = sStore.UpdateServiceAlias(&model.Service{
		ID:         "svcid2",
		Name:       "svcname1",
		Namespace:  "testsvc",
		Owner:      "testo",
		Token:      "t1",
		Revision:   "modifyRevision2",
		Reference:  "m1",
		Business:   "modifyBusiness",
		Department: "modifyDepartment",
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

	if ss.Business != "modifyBusiness" ||
		ss.Department != "modifyDepartment" ||
		ss.Revision != "modifyRevision2" {
		t.Fatal("update service error")
	}
}

func TestServiceStore_DeleteServiceAlias(t *testing.T) {
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

	svc, err := sStore.getServiceByNameAndNs("svcname0", "testsvc")
	assert.Nil(t, err, "error must be nil")

	assert.False(t, svc.Valid, "delete service alias failed")
}

func TestServiceStore_DeleteService(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}

	defer handler.Close()

	sStore := &serviceStore{handler: handler}

	_, ss, err := sStore.GetServices(nil, nil, nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range ss {
		fmt.Printf("get service %+v\n", s)
		err := sStore.DeleteService(s.ID, "", "")
		if err != nil {
			t.Fatal(err)
		}
	}

	// check delete res
	total, s, err := sStore.GetServices(nil, nil, nil, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatal("delete service not effect")
	}

	for _, val := range s {
		assert.False(t, val.Valid, "delete service not effect")
	}

}
