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
	"errors"
	"fmt"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

// DefaultData 默认数据信息
type DefaultData struct {
	Namespaces []*model.Namespace           `yaml:"namespaces"`
	Users      []*authcommon.User           `yaml:"users"`
	Policies   []*authcommon.StrategyDetail `yaml:"policies"`
}

func (m *boltStore) loadByFile(loadFile string) error {
	if len(loadFile) == 0 {
		return errors.New("loadFile is empty")
	}
	cf, err := os.Open(loadFile)
	if err != nil {
		// 如果文件不存在，则需要用户首次调用接口进行管理员帐户的初始化操作
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer cf.Close()
	data := &DefaultData{}
	if err := yaml.NewDecoder(cf).Decode(data); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}
	if len(data.Namespaces) == 0 {
		// 降级走默认配置
		if err := m.initNamingStoreData(); err != nil {
			_ = m.handler.Close()
			return err
		}
	}
	return m.loadFromData(data)
}

func (m *boltStore) loadFromData(data *DefaultData) error {
	if err := m.handler.Execute(true, func(tx *bolt.Tx) error {
		for i := range data.Users {
			saveUser, err := m.getUser(tx, data.Users[i].ID)
			if err != nil {
				return err
			}
			if saveUser == nil {
				data.Users[i].CreateTime = time.Now()
				data.Users[i].ModifyTime = time.Now()
				// 添加主账户主体信息
				if err := saveValue(tx, tblUser, data.Users[i].ID, converToUserStore(data.Users[i])); err != nil {
					log.Error("[Store][User] save user fail", zap.Error(err), zap.String("name", data.Users[i].Name))
					return err
				}
			}
		}

		for i := range data.Policies {
			saveRule, err := m.getStrategyDetail(tx, data.Policies[i].ID)
			if err != nil {
				return err
			}

			if saveRule == nil {
				data.Policies[i].CreateTime = time.Now()
				data.Policies[i].ModifyTime = time.Now()
				// 添加主账户的默认鉴权策略信息
				if err := saveValue(tx, tblStrategy, data.Policies[i].ID, convertForStrategyStore(data.Policies[i])); err != nil {
					log.Error("[Store][Strategy] save auth_strategy", zap.Error(err),
						zap.String("name", data.Policies[i].Name), zap.String("owner", data.Policies[i].Owner))
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
