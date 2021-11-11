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
	"github.com/polarismesh/polaris-server/common/model"
	"strconv"
	"testing"
	"time"
)

const (
	nsCount   = 5
	nsOwner   = "user"
	nsComment = "test ns"
	nsToken   = "xxxxx"
)

func TestNamespaceStore_AddNamespace(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	for i := 0; i < nsCount; i++ {
		err = nsStore.AddNamespace(&model.Namespace{
			Name:       "default" + strconv.Itoa(i),
			Comment:    nsComment,
			Token:      nsToken,
			Owner:      nsOwner,
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if nil != err {
			t.Fatal(err)
		}
	}
}

func TestNamespaceStore_ListNamespaces(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	namespaces, err := nsStore.ListNamespaces(nsOwner)
	if nil != err {
		t.Fatal(err)
	}
	for _, namespace := range namespaces {
		fmt.Printf("namespace is %+v\n", namespace)
	}
	if len(namespaces) != nsCount {
		t.Fatal(fmt.Sprintf("namespaces count not match, expect %d, got %d", nsCount, len(namespaces)))
	}
}

func TestNamespaceStore_GetNamespace(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if nil != err {
			t.Fatal(err)
		}
		if nil == ns {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
	}
}

func TestNamespaceStore_UpdateNamespace(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	for i := 0; i < nsCount; i++ {
		nsRaw := &model.Namespace{
			Name:    "default" + strconv.Itoa(i),
			Comment: nsComment + strconv.Itoa(i),
			Owner:   nsOwner,
		}
		err = nsStore.UpdateNamespace(nsRaw)
		if nil != err {
			t.Fatal(err)
		}
	}
	//检查update是否生效
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if nil != err {
			t.Fatal(err)
		}
		if nil == ns {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
		if ns.Comment != nsComment+strconv.Itoa(i) {
			t.Fatal(fmt.Sprintf("comment not updated for %s", name))
		}
	}
}

func TestNamespaceStore_UpdateNamespaceToken(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		token := nsToken + strconv.Itoa(i)
		err = nsStore.UpdateNamespaceToken(name, token)
		if nil != err {
			t.Fatal(err)
		}
	}
	//检查update是否生效
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if nil != err {
			t.Fatal(err)
		}
		if nil == ns {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
		if ns.Token != nsToken+strconv.Itoa(i) {
			t.Fatal(fmt.Sprintf("comment not updated for %s", name))
		}
	}
}

func TestNamespaceStore_GetMoreNamespaces(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	before := time.Now().Add(0 - 1*time.Minute)
	namespaces, err := nsStore.GetMoreNamespaces(before)
	if nil != err {
		t.Fatal(err)
	}
	if len(namespaces) != nsCount {
		t.Fatal(fmt.Sprintf("more namespaces count not match, expect %d, got %d", nsCount, len(namespaces)))
	}
}

func TestTransaction_LockNamespace(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	trans := &transaction{handler: handler}
	defer trans.Commit()
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		namespace, err := trans.LockNamespace(name)
		if nil != err {
			t.Fatal(err)
		}
		if nil == namespace {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
	}
}

func TestTransaction_DeleteNamespace(t *testing.T) {
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if nil != err {
		t.Fatal(err)
	}
	defer handler.Close()
	trans := &transaction{handler: handler}
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		err := trans.DeleteNamespace(name)
		if nil != err {
			trans.Commit()
			t.Fatal(err)
		}
	}
	err = trans.Commit()
	if nil != err {
		t.Fatal(err)
	}
	nsStore := &namespaceStore{handler: handler}
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if nil != err {
			t.Fatal(err)
		}
		if nil != ns {
			t.Fatal(fmt.Sprintf("namespace %s exists, delete fail", name))
		}
	}
}
