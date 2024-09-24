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
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.GrayStore = (*grayStore)(nil)

const (
	tblGrayResource             string = "GrayResource"
	GrayResourceFieldModifyTime string = "ModifyTime"
)

type grayStore struct {
	handler BoltHandler
}

func newGrayStore(handler BoltHandler) *configFileReleaseStore {
	s := &configFileReleaseStore{handler: handler}
	return s
}

// CreateGrayResourceTx 新建灰度资源
func (cfr *grayStore) CreateGrayResourceTx(proxyTx store.Tx, grayResource *model.GrayResource) error {
	tx := proxyTx.GetDelegateTx().(*bolt.Tx)

	tN := time.Now()
	grayResource.CreateTime = tN
	grayResource.ModifyTime = tN

	err := saveValue(tx, tblGrayResource, grayResource.Name, grayResource)
	if err != nil {
		log.Error("[GrayResource] save info", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

func (cfr *grayStore) CleanGrayResource(proxyTx store.Tx, data *model.GrayResource) error {
	tx := proxyTx.GetDelegateTx().(*bolt.Tx)

	properties := map[string]interface{}{
		GrayResourceFieldModifyTime: time.Now(),
		CommonFieldValid:            false,
	}

	err := updateValue(tx, tblGrayResource, data.Name, properties)
	if err != nil {
		log.Error("[GrayResource] update info", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// GetMoreGrayResouces Get the last update time more than a certain time point
func (cfr *grayStore) GetMoreGrayResouces(firstUpdate bool,
	modifyTime time.Time) ([]*model.GrayResource, error) {

	if firstUpdate {
		modifyTime = time.Time{}
	}

	fields := []string{GrayResourceFieldModifyTime}
	values, err := cfr.handler.LoadValuesByFilter(tblGrayResource, fields, &model.GrayResource{},
		func(m map[string]interface{}) bool {
			saveMt, _ := m[GrayResourceFieldModifyTime].(time.Time)
			return !saveMt.Before(modifyTime)
		})

	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return []*model.GrayResource{}, nil
	}

	grayResources := make([]*model.GrayResource, 0, len(values))

	for i := range values {
		grayResource := values[i].(*model.GrayResource)
		grayResources = append(grayResources, grayResource)
	}
	return grayResources, nil

}
