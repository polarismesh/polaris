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

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/stretchr/testify/assert"
)

func mockConfigFileRelease(total int) []*model.ConfigFileRelease {

	ret := make([]*model.ConfigFileRelease, 0, total)

	for i := 0; i < total; i++ {
		ret = append(ret, &model.ConfigFileRelease{
			Name:       fmt.Sprintf("config-file-release-%d", i),
			Namespace:  fmt.Sprintf("config-file-release-%d", i),
			Group:      fmt.Sprintf("config-file-release-%d", i),
			FileName:   fmt.Sprintf("config-file-release-%d", i),
			Content:    fmt.Sprintf("config-file-release-%d", i),
			Comment:    fmt.Sprintf("config-file-release-%d", i),
			Md5:        fmt.Sprintf("config-file-release-%d", i),
			Version:    0,
			Flag:       0,
			CreateTime: time.Time{},
			CreateBy:   "",
			ModifyTime: time.Time{},
			ModifyBy:   "",
			Valid:      false,
		})
	}

	return ret
}

func Test_configFileReleaseStore(t *testing.T) {
	t.Run("创建配置Release-不带事务", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				cfr, err := s.CreateConfigFileRelease(nil, ret[i])

				assert.NoError(t, err, err)

				assert.Equal(t, uint64(i+1), cfr.Id)
			}
		})
	})

	t.Run("创建配置Release-带事务", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				func() {
					tx, err := handler.StartTx()
					assert.NoError(t, err, err)

					defer tx.Rollback()

					cfr, err := s.CreateConfigFileRelease(tx, ret[i])
					assert.NoError(t, err, err)
					assert.Equal(t, uint64(i+1), cfr.Id)

					err = tx.Commit()
					assert.NoError(t, err, err)
				}()
			}
		})
	})

	t.Run("更新配置Release", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				cfr, err := s.CreateConfigFileRelease(nil, ret[i])

				assert.NoError(t, err, err)
				assert.Equal(t, uint64(i+1), cfr.Id)

				cfr.Comment = "update config release"
				cfr.Content = "update config release"

				newCfr, err := s.UpdateConfigFileRelease(nil, cfr)

				assert.NoError(t, err, err)
				assert.Equal(t, uint64(i+1), newCfr.Id)
				assert.Equal(t, cfr.Content, newCfr.Content)
				assert.Equal(t, cfr.Comment, newCfr.Comment)
			}
		})
	})

	t.Run("删除配置Release", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				cfr, err := s.CreateConfigFileRelease(nil, ret[i])

				assert.NoError(t, err, err)
				assert.Equal(t, uint64(i+1), cfr.Id)

				err = s.DeleteConfigFileRelease(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName, "")
				assert.NoError(t, err, err)

				oldCfr, err := s.GetConfigFileRelease(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName)
				assert.NoError(t, err, err)
				assert.Nil(t, oldCfr)
			}
		})
	})

	t.Run("删除配置Release-可以查询逻辑删除的数据", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				cfr, err := s.CreateConfigFileRelease(nil, ret[i])

				assert.NoError(t, err, err)
				assert.Equal(t, uint64(i+1), cfr.Id)

				saveCfr, err := s.GetConfigFileRelease(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName)
				assert.NoError(t, err, err)
				assert.NotNil(t, saveCfr)

				err = s.DeleteConfigFileRelease(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName, "")
				assert.NoError(t, err, err)

				oldCfr, err := s.GetConfigFileRelease(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName)
				assert.NoError(t, err, err)
				assert.Nil(t, oldCfr)

				oldCfr, err = s.GetConfigFileReleaseWithAllFlag(nil, ret[i].Namespace, ret[i].Group, ret[i].FileName)
				assert.NoError(t, err, err)
				assert.NotNil(t, oldCfr)
				assert.False(t, oldCfr.Valid)
				assert.Equal(t, 1, oldCfr.Flag)

				saveCfr.Id = 0
				saveCfr.CreateTime = time.Time{}
				saveCfr.ModifyTime = time.Time{}
				saveCfr.Flag = oldCfr.Flag
				saveCfr.Valid = oldCfr.Valid
				saveCfr.Md5 = oldCfr.Md5
				saveCfr.Version = oldCfr.Version

				oldCfr.Id = 0
				oldCfr.CreateTime = time.Time{}
				oldCfr.ModifyTime = time.Time{}

				assert.Equal(t, saveCfr, oldCfr, "saveCfr : %#v, oldCfr : %#v", saveCfr, oldCfr)
			}
		})
	})

	t.Run("查询Release信息-用于刷新Cache缓存", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(10)

			save := make([]*model.ConfigFileRelease, 0, len(ret))

			for i := range ret {
				cfr, err := s.CreateConfigFileRelease(nil, ret[i])
				assert.NoError(t, err, err)
				assert.Equal(t, uint64(i+1), cfr.Id)

				save = append(save, cfr)
			}

			result, err := s.FindConfigFileReleaseByModifyTimeAfter(time.Time{})
			assert.NoError(t, err, err)

			assert.ElementsMatch(t, save, result, fmt.Sprintf("expect %#v, actual %#v", save, result))

			result, err = s.FindConfigFileReleaseByModifyTimeAfter(time.Now().Add(time.Duration(1 * time.Hour)))
			assert.NoError(t, err, err)
			assert.Empty(t, result)
		})
	})
}
