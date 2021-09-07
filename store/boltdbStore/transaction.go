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

package boltdbStore

import (
	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
)

type transaction struct {
	tx *bolt.Tx
}

// 提交事务
func (t *transaction) Commit() error {
	return t.tx.Commit()
}

// 启动锁，限制Server启动的并发数
func (t *transaction) LockBootstrap(key string, server string) error {
	return nil
}

// 排它锁namespace
func (t *transaction) LockNamespace(name string) (*model.Namespace, error) {
	return nil, nil
}

// 共享锁namespace
func (t *transaction) RLockNamespace(name string) (*model.Namespace, error) {
	return nil, nil
}

// 删除namespace
func (t *transaction) DeleteNamespace(name string) error {
	return nil
}

// 排它锁service
func (t *transaction) LockService(name string, namespace string) (*model.Service, error) {
	return nil, nil
}

// 共享锁service
func (t *transaction) RLockService(name string, namespace string) (*model.Service, error) {
	return nil, nil
}

// 批量锁住service，只需返回valid/bool，增加速度
func (t *transaction) BatchRLockServices(ids map[string]bool) (map[string]bool, error) {
	return nil, nil
}

// 删除service
func (t *transaction) DeleteService(name string, namespace string) error {
	return nil
}

// 删除源服服务下的所有别名
func (t *transaction) DeleteAliasWithSourceID(sourceServiceID string) error {
	return nil
}
