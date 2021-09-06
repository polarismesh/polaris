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
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

type namespaceStore struct {
	handler BoltHandler
}

// 保存一个命名空间
func (n *namespaceStore) AddNamespace(namespace *model.Namespace) error {
	//TODO
	return nil
}

// 更新命名空间
func (n *namespaceStore) UpdateNamespace(namespace *model.Namespace) error {
	//TODO
	return nil
}

// 更新命名空间token
func (n *namespaceStore) UpdateNamespaceToken(name string, token string) error {
	//TODO
	return nil
}

// 查询owner下所有的命名空间
func (n *namespaceStore) ListNamespaces(owner string) ([]*model.Namespace, error) {
	//TODO
	return nil, nil
}

// 根据name获取命名空间的详情
func (n *namespaceStore) GetNamespace(name string) (*model.Namespace, error) {
	//TODO
	return nil, nil
}

// 从数据库查询命名空间
func (n *namespaceStore) GetNamespaces(
	filter map[string][]string, offset, limit int) ([]*model.Namespace, uint32, error) {
	//TODO
	return nil, 0, nil
}

// 获取增量数据
func (n *namespaceStore) GetMoreNamespaces(mtime time.Time) ([]*model.Namespace, error) {
	//TODO
	return nil, nil
}