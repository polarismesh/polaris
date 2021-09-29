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
	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
)

type transaction struct {
	tx  *bolt.Tx
	err error
}

// Commit 提交事务
func (t *transaction) Commit() error {
	if nil != t.err {
		return t.tx.Rollback()
	}
	err := t.tx.Commit()
	if nil != err {
		return t.tx.Rollback()
	}
	return nil
}

// LockBootstrap 启动锁，限制Server启动的并发数
func (t *transaction) LockBootstrap(key string, server string) error {
	return nil
}

// LockNamespace 排它锁namespace
func (t *transaction) LockNamespace(name string) (*model.Namespace, error) {
	return t.loadNamespace(name)
}

func (t *transaction) loadNamespace(name string) (*model.Namespace, error) {
	var values = make(map[string]interface{})
	t.err = loadValues(t.tx, tblNameNamespace, []string{name}, &model.Namespace{}, values)
	if nil != t.err {
		return nil, t.err
	}
	value, ok := values[name]
	if !ok {
		return nil, nil
	}
	return value.(*model.Namespace), nil
}

// RLockNamespace 共享锁namespace
func (t *transaction) RLockNamespace(name string) (*model.Namespace, error) {
	return t.loadNamespace(name)
}

// DeleteNamespace 删除namespace
func (t *transaction) DeleteNamespace(name string) error {
	t.err = deleteValues(t.tx, tblNameNamespace, []string{name})
	if nil != t.err {
		return t.err
	}
	return nil
}

const (
	svcFieldName      = "Name"
	svcFieldNamespace = "Namespace"
)

func (t *transaction) loadService(name string, namespace string) (*model.Service, error) {
	filter := func(m map[string]interface{}) bool {
		nameValue, ok := m[svcFieldName]
		if !ok {
			return false
		}
		namespaceValue, ok := m[svcFieldNamespace]
		if !ok {
			return false
		}
		return nameValue.(string) == name && namespaceValue.(string) == namespace
	}
	values := make(map[string]interface{})
	err := loadValuesByFilter(
		t.tx, tblNameService, []string{svcFieldName, svcFieldNamespace}, &model.Service{}, filter, values)
	if nil != err {
		return nil, err
	}
	var svc *model.Service
	for _, svcValue := range values {
		svc = svcValue.(*model.Service)
		break
	}
	return svc, nil
}

// LockService 排它锁service
func (t *transaction) LockService(name string, namespace string) (*model.Service, error) {
	return t.loadService(name, namespace)
}

// RLockService 共享锁service
func (t *transaction) RLockService(name string, namespace string) (*model.Service, error) {
	return t.loadService(name, namespace)
}
