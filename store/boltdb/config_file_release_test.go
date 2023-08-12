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

func mockConfigFileRelease(total int) []*model.ConfigFileRelease {

	ret := make([]*model.ConfigFileRelease, 0, total)

	for i := 0; i < total; i++ {
		ret = append(ret, &model.ConfigFileRelease{
			SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
				ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
					Name:      fmt.Sprintf("config-file-release-%d", i),
					Namespace: fmt.Sprintf("config-file-release-%d", i),
					Group:     fmt.Sprintf("config-file-release-%d", i),
					FileName:  fmt.Sprintf("config-file-release-%d", i),
				},
				Comment:    fmt.Sprintf("config-file-release-%d", i),
				Md5:        fmt.Sprintf("config-file-release-%d", i),
				Version:    0,
				Flag:       0,
				CreateTime: time.Time{},
				CreateBy:   "",
				ModifyTime: time.Time{},
				ModifyBy:   "",
				Valid:      false,
			},
			Content: fmt.Sprintf("config-file-release-%d", i),
		})
	}

	return ret
}

func Test_configFileReleaseStore(t *testing.T) {
	t.Run("创建配置Release", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				func() {
					tx, err := handler.StartTx()
					assert.NoError(t, err, err)
					defer tx.Rollback()

					err = s.CreateConfigFileReleaseTx(tx, ret[i])
					assert.NoError(t, err, err)
					err = tx.Commit()
					assert.NoError(t, err, err)
				}()
			}
		})
	})

	t.Run("删除配置Release", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(1)

			for i := range ret {
				tx, err := handler.StartTx()
				assert.NoError(t, err, err)
				defer tx.Rollback()

				err = s.CreateConfigFileReleaseTx(tx, ret[i])
				assert.NoError(t, err, err)

				searchKey := &model.ConfigFileReleaseKey{
					Namespace: ret[i].Namespace,
					Group:     ret[i].Group,
					FileName:  ret[i].FileName,
					Name:      ret[i].Name,
				}
				err = s.DeleteConfigFileReleaseTx(tx, searchKey)
				assert.NoError(t, err, err)

				oldCfr, err := s.GetConfigFileRelease(searchKey)
				assert.NoError(t, err, err)
				assert.Nil(t, oldCfr)

				err = tx.Commit()
				assert.NoError(t, err, err)
			}
		})
	})

	t.Run("查询Release信息-用于刷新Cache缓存", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileRelease, func(t *testing.T, handler BoltHandler) {

			s := &configFileReleaseStore{handler: handler}

			ret := mockConfigFileRelease(10)

			save := make([]*model.ConfigFileRelease, 0, len(ret))

			for i := range ret {
				tx, err := handler.StartTx()
				assert.NoError(t, err, err)
				defer tx.Rollback()

				err = s.CreateConfigFileReleaseTx(tx, ret[i])
				assert.NoError(t, err, err)
				err = tx.Commit()
				assert.NoError(t, err, err)
				save = append(save, ret[i])
			}

			result, err := s.GetMoreReleaseFile(true, time.Time{})
			assert.NoError(t, err, err)
			assert.Equal(t, int(len(save)), int(len(result)))

			result, err = s.GetMoreReleaseFile(false, time.Now().Add(time.Duration(1*time.Hour)))
			assert.NoError(t, err, err)
			assert.Empty(t, result)
		})
	})
}
