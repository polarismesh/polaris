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

type l5Store struct {
	handler BoltHandler
}

// 获取扩展数据
func (l *l5Store) GetL5Extend(serviceID string) (map[string]interface{}, error) {
	//TODO
	return nil, nil
}

// 设置meta里保存的扩展数据，并返回剩余的meta
func (l *l5Store) SetL5Extend(serviceID string, meta map[string]interface{}) (map[string]interface{}, error) {
	//TODO
	return nil, nil
}

// 获取module
func (l *l5Store) GenNextL5Sid(layoutID uint32) (string, error) {
	//TODO
	return "", nil
}

// 获取增量数据
func (l *l5Store) GetMoreL5Extend(mtime time.Time) (map[string]map[string]interface{}, error) {
	//TODO
	return nil, nil
}

// 获取Route增量数据
func (l *l5Store) GetMoreL5Routes(flow uint32) ([]*model.Route, error) {
	//TODO
	return nil, nil
}

// 获取Policy增量数据
func (l *l5Store) GetMoreL5Policies(flow uint32) ([]*model.Policy, error) {
	//TODO
	return nil, nil
}

//获取Section增量数据
func (l *l5Store) GetMoreL5Sections(flow uint32) ([]*model.Section, error) {
	//TODO
	return nil, nil
}

//获取IP Config增量数据
func (l *l5Store) GetMoreL5IPConfigs(flow uint32) ([]*model.IPConfig, error) {
	//TODO
	return nil, nil
}
