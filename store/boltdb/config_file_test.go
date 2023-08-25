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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

func mockConfigFile(total int, param map[string]string) []*model.ConfigFile {
	ret := make([]*model.ConfigFile, 0, total)

	for i := 0; i < total; i++ {

		namespace := param["namespace"]
		group := param["group"]

		if namespace == "" {
			namespace = fmt.Sprintf("cpnfig-file-%d", i)
		}

		if group == "" {
			group = fmt.Sprintf("cpnfig-file-%d", i)
		}

		ret = append(ret, &model.ConfigFile{
			Id:         0,
			Name:       fmt.Sprintf("cpnfig-file-%d", i),
			Namespace:  namespace,
			Group:      group,
			Content:    fmt.Sprintf("cpnfig-file-%d", i),
			Comment:    fmt.Sprintf("cpnfig-file-%d", i),
			Format:     "yaml",
			Flag:       0,
			CreateTime: time.Time{},
			CreateBy:   "polaris",
			ModifyTime: time.Time{},
			ModifyBy:   "polaris",
			Valid:      false,
			Metadata:   map[string]string{},
		})
	}

	return ret
}

func Test_configFileStore(t *testing.T) {
	t.Run("创建配置文件-无事务", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFile, func(t *testing.T, handler BoltHandler) {

			s := &configFileStore{handler: handler}

			mocks := mockConfigFile(10, map[string]string{})

			for i := range mocks {
				waitSave := mocks[i]
				tx, err := s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				defer func() {
					_ = tx.Rollback()
				}()
				err = s.CreateConfigFileTx(tx, waitSave)
				assert.NoError(t, err, "%+v", err)
				err = tx.Commit()
				assert.NoError(t, err, "%+v", err)
			}
		})
	})

	t.Run("查询配置文件", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFile, func(t *testing.T, handler BoltHandler) {

			s := &configFileStore{handler: handler}

			mocks := mockConfigFile(10, map[string]string{})

			for i := range mocks {

				waitSave := mocks[i]
				tx, err := s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				defer func() {
					_ = tx.Rollback()
				}()
				err = s.CreateConfigFileTx(tx, waitSave)
				assert.NoError(t, err, "%+v", err)
				err = tx.Commit()
				assert.NoError(t, err, "%+v", err)

				r, err := s.GetConfigFile(waitSave.Namespace, waitSave.Group, waitSave.Name)
				assert.NoError(t, err, "%+v", err)
				assert.NotNil(t, r)
			}
		})
	})

	t.Run("删除配置文件", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFile, func(t *testing.T, handler BoltHandler) {

			s := &configFileStore{handler: handler}

			mocks := mockConfigFile(10, map[string]string{})

			for i := range mocks {

				waitSave := mocks[i]
				tx, err := s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				err = s.CreateConfigFileTx(tx, waitSave)
				assert.NoError(t, err, "%+v", err)
				_ = tx.Commit()

				tx, err = s.handler.StartTx()
				err = s.DeleteConfigFileTx(tx, waitSave.Namespace, waitSave.Group, waitSave.Name)
				assert.NoError(t, err, "%+v", err)
				_ = tx.Commit()

				tx, err = s.handler.StartTx()
				r, err := s.GetConfigFileTx(tx, waitSave.Namespace, waitSave.Group, waitSave.Name)
				assert.NoError(t, err, "%+v", err)
				assert.Nil(t, r)
				_ = tx.Commit()
			}
		})
	})

	t.Run("更新配置文件", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFile, func(t *testing.T, handler BoltHandler) {

			s := &configFileStore{handler: handler}

			mocks := mockConfigFile(10, map[string]string{})

			for i := range mocks {

				waitSave := mocks[i]
				tx, err := s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				err = s.CreateConfigFileTx(tx, waitSave)
				assert.NoError(t, err, "%+v", err)
				_ = tx.Commit()

				newCf := *waitSave

				newCf.Comment = "update config file"

				tx, err = s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				err = s.UpdateConfigFileTx(tx, &newCf)
				assert.NoError(t, err, "%+v", err)
				_ = tx.Commit()

				r, err := s.GetConfigFile(waitSave.Namespace, waitSave.Group, waitSave.Name)
				assert.NoError(t, err, "%+v", err)

				_n := &newCf

				r.CreateTime = time.Unix(0, 0)
				r.ModifyTime = time.Unix(0, 0)
				r.ReleaseTime = time.Unix(0, 0)
				_n.CreateTime = time.Unix(0, 0)
				_n.ModifyTime = time.Unix(0, 0)
				_n.ReleaseTime = time.Unix(0, 0)
				assert.Equal(t, _n, r, "expect : %#v, actual : %#v", _n, r)
			}
		})
	})

	t.Run("查询配置文件", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFile, func(t *testing.T, handler BoltHandler) {

			s := &configFileStore{handler: handler}

			mocks := mockConfigFile(10, map[string]string{})
			results := make([]*model.ConfigFile, 0, len(mocks))

			for i := range mocks {
				waitSave := mocks[i]
				tx, err := s.handler.StartTx()
				assert.NoError(t, err, "%+v", err)
				err = s.CreateConfigFileTx(tx, waitSave)
				assert.NoError(t, err, "%+v", err)
				_ = tx.Commit()

				assert.NoError(t, err, "%+v", err)
				results = append(results, waitSave)
			}

			total, ret, err := s.QueryConfigFiles(map[string]string{
				"namespace": "cpnfig*",
				"group":     "cpnfig*",
				"name":      "cpnfig*",
			}, 0, 100)

			for i := range ret {
				ret[i].CreateTime = time.Time{}
				ret[i].ModifyTime = time.Time{}
				results[i].CreateTime = time.Time{}
				results[i].ModifyTime = time.Time{}
			}

			assert.NoError(t, err, "%+v", err)
			assert.Equal(t, len(results), len(ret))
			assert.Equal(t, len(mocks), int(total))

			total, ret, err = s.QueryConfigFiles(map[string]string{
				"namespace": "qweq",
				"group":     "qweq",
				"name":      "qweq",
			}, 0, 100)

			for i := range ret {
				ret[i].CreateTime = time.Time{}
				ret[i].ModifyTime = time.Time{}
				results[i].CreateTime = time.Time{}
				results[i].ModifyTime = time.Time{}
			}

			assert.NoError(t, err, "%+v", err)
			assert.Equal(t, 0, int(total))
			assert.Empty(t, ret)
		})
	})
}
