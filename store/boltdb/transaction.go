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
	"github.com/polarismesh/polaris/common/model"
)

type transaction struct {
	handler BoltHandler
}

// Commit 提交事务
func (t *transaction) Commit() error {
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
	values, err := t.handler.LoadValues(tblNameNamespace, []string{name}, &Namespace{})
	if err != nil {
		return nil, err
	}
	value, ok := values[name]
	if !ok {
		return nil, nil
	}
	return toModelNamespace(value.(*Namespace)), nil
}

// RLockNamespace 共享锁namespace
func (t *transaction) RLockNamespace(name string) (*model.Namespace, error) {
	return t.loadNamespace(name)
}

// DeleteNamespace 删除namespace
func (t *transaction) DeleteNamespace(name string) error {
	return t.handler.DeleteValues(tblNameNamespace, []string{name})
}

const (
	svcFieldName      string = "Name"
	svcFieldNamespace string = "Namespace"
	svcFieldValid     string = "Valid"
)

func (t *transaction) loadService(name string, namespace string) (*model.Service, error) {
	filter := func(m map[string]interface{}) bool {
		validVal, ok := m[svcFieldValid]
		if ok && !validVal.(bool) {
			return false
		}
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
	values, err := t.handler.LoadValuesByFilter(
		tblNameService, []string{svcFieldName, svcFieldNamespace, svcFieldValid}, &Service{}, filter)
	if err != nil {
		return nil, err
	}
	var svc *model.Service
	for _, svcValue := range values {
		svc = toModelService(svcValue.(*Service))
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
