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
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/polarismesh/polaris/common/model"
)

const (
	nsCount   = 5
	nsOwner   = "user"
	nsComment = "test ns"
	nsToken   = "xxxxx"
)

func InitNamespaceData(nsStore *namespaceStore, nsCount int) error {
	for i := 0; i < nsCount; i++ {
		err := nsStore.AddNamespace(&model.Namespace{
			Name:       "default" + strconv.Itoa(i),
			Comment:    nsComment,
			Token:      nsToken,
			Owner:      nsOwner,
			Valid:      true,
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func TestNamespaceStore_AddNamespace(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		handler.Close()
	}()
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
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNamespaceStore_GetNamespaces(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}

	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	t.Run("正常查询命名空间列表", func(t *testing.T) {
		ret, retCnt, err := nsStore.GetNamespaces(map[string][]string{
			"": {},
		}, 0, nsCount)

		if err != nil {
			t.Fatal(err)
		}
		if len(ret) != int(retCnt) {
			t.Fatal("len(ret) need equal int(retCnt)")
		}
	})

	t.Run("查询条件不满足-无法查询出结果", func(t *testing.T) {
		// 只要有一个条件不满足，则对应的条目就不应该查出来
		ret, _, err := nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {"springliao"},
		}, 0, nsCount)

		if err != nil {
			t.Fatal(err)
		}
		if len(ret) != 0 {
			t.Fatal("len(ret) must be zero")
		}

		ret, _, err = nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
			NameAttribute:  {"springliao"},
		}, 0, nsCount)

		if err != nil {
			t.Fatal(err)
		}
		if len(ret) != 0 {
			t.Fatal("len(ret) must be zero")
		}
	})

	t.Run("条件分页-offset为0开始查询-只查一条数据", func(t *testing.T) {
		ret, retCnt, err := nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
		}, 0, 1)

		if err != nil {
			t.Fatal(err)
		}
		if !(len(ret) == 1 && nsCount == int(retCnt)) {
			t.Fatalf("len(ret) must be 1 and retCnt must be %d", nsCount)
		}

		ret, retCnt, err = nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
			NameAttribute:  {"default1"},
		}, 0, 1)

		if err != nil {
			t.Fatal(err)
		}
		if !(len(ret) == 1 && retCnt == 1) {
			t.Fatalf("len(ret) must be 1 and retCnt must be 1, acutal len(ret) %d, retCnt : %d", len(ret), retCnt)
		}

	})

	t.Run("条件分页查询-从offset3开始查询-只查一条数据", func(t *testing.T) {
		ret, retCnt, err := nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
		}, 3, 1)

		if err != nil {
			t.Fatal(err)
		}
		if !(len(ret) == 1 && nsCount == int(retCnt)) {
			t.Fatalf("len(ret) must be 1 and retCnt must be %d", nsCount)
		}
	})

	t.Run("条件分页查询-查询多条数据", func(t *testing.T) {
		ret, retCnt, err := nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
		}, 3, 10)

		if err != nil {
			t.Fatal(err)
		}
		if !(len(ret) == 2 && nsCount == int(retCnt)) {
			t.Fatalf("len(ret) must be 1 and retCnt must be %d", nsCount)
		}
	})

	t.Run("分页查询-offset太大", func(t *testing.T) {
		ret, retCnt, err := nsStore.GetNamespaces(map[string][]string{
			OwnerAttribute: {nsOwner},
		}, 1000000, 10)

		if err != nil {
			t.Fatal(err)
		}
		if !(len(ret) == 0 && nsCount == int(retCnt)) {
			t.Fatalf("len(ret) must be 1 and retCnt must be %d", nsCount)
		}
	})

}

func TestNamespaceStore_GetNamespace(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}

	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if err != nil {
			t.Fatal(err)
		}
		if ns == nil {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
	}
}

func TestNamespaceStore_UpdateNamespace(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}

	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < nsCount; i++ {
		nsRaw := &model.Namespace{
			Name:    "default" + strconv.Itoa(i),
			Comment: nsComment + strconv.Itoa(i),
			Owner:   nsOwner,
		}
		err = nsStore.UpdateNamespace(nsRaw)
		if err != nil {
			t.Fatal(err)
		}
	}
	// 检查update是否生效
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if err != nil {
			t.Fatal(err)
		}
		if ns == nil {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
		if ns.Comment != nsComment+strconv.Itoa(i) {
			t.Fatal(fmt.Sprintf("comment not updated for %s", name))
		}
	}
}

func TestNamespaceStore_UpdateNamespaceToken(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}

	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		token := nsToken + strconv.Itoa(i)
		err = nsStore.UpdateNamespaceToken(name, token)
		if err != nil {
			t.Fatal(err)
		}
	}
	// 检查update是否生效
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if err != nil {
			t.Fatal(err)
		}
		if ns == nil {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
		if ns.Token != nsToken+strconv.Itoa(i) {
			t.Fatal(fmt.Sprintf("comment not updated for %s", name))
		}
	}
}

func TestNamespaceStore_GetMoreNamespaces(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	nsStore := &namespaceStore{handler: handler}
	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	before := time.Now().Add(0 - 1*time.Minute)
	namespaces, err := nsStore.GetMoreNamespaces(before)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != nsCount {
		t.Fatal(fmt.Sprintf("more namespaces count not match, expect %d, got %d", nsCount, len(namespaces)))
	}
}

func TestTransaction_LockNamespace(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	trans := &transaction{handler: handler}

	if err := InitNamespaceData(&namespaceStore{handler: handler}, nsCount); err != nil {
		t.Fatal(err)
	}

	defer trans.Commit()
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		namespace, err := trans.LockNamespace(name)
		if err != nil {
			t.Fatal(err)
		}
		if namespace == nil {
			t.Fatal(fmt.Sprintf("namespace %s not exists", name))
		}
	}
}

func TestTransaction_DeleteNamespace(t *testing.T) {
	_ = os.RemoveAll("./table.bolt")
	handler, err := NewBoltHandler(&BoltConfig{FileName: "./table.bolt"})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	trans := &transaction{handler: handler}

	nsStore := &namespaceStore{handler: handler}
	if err := InitNamespaceData(nsStore, nsCount); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		err := trans.DeleteNamespace(name)
		if err != nil {
			trans.Commit()
			t.Fatal(err)
		}
	}
	err = trans.Commit()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < nsCount; i++ {
		name := "default" + strconv.Itoa(i)
		ns, err := nsStore.GetNamespace(name)
		if err != nil {
			t.Fatal(err)
		}
		if ns != nil {
			t.Fatal(fmt.Sprintf("namespace %s exists, delete fail", name))
		}
	}
}
