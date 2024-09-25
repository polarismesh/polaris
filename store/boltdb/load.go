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
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

func (m *boltStore) loadByDefault() error {
	if err := m.initAuthStoreData(); err != nil {
		_ = m.handler.Close()
		return err
	}
	if err := m.initNamingStoreData(); err != nil {
		_ = m.handler.Close()
		return err
	}
	return nil
}

// DefaultData 默认数据信息
type DefaultData struct {
	Namespaces []*model.Namespace `yaml:"namespaces"`
	Users      []*authcommon.User `yaml:"users"`
}

func (m *boltStore) loadByFile(loadFile string) error {
	if len(loadFile) == 0 {
		return errors.New("loadFile is empty")
	}
	cf, err := os.Open(loadFile)
	if err != nil {
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
	if len(data.Users) == 0 {
		if err := m.initAuthStoreData(); err != nil {
			_ = m.handler.Close()
			return err
		}
		return nil
	}
	return m.loadFromData(data)
}

func (m *boltStore) loadFromData(data *DefaultData) error {
	users := data.Users

	// 确保排序为 admin -> main -> sub
	sort.Slice(users, func(i, j int) bool {
		return users[i].Type < users[j].Type
	})

	tn := time.Now()
	var (
		superUser, mainUser *authcommon.User
	)
	if len(users) >= 2 && users[0].Type == authcommon.AdminUserRole && users[1].Type == authcommon.OwnerUserRole {
		superUser = users[0]
		superUser.CreateTime = tn
		superUser.ModifyTime = tn
		mainUser = users[1]
		mainUser.CreateTime = tn
		mainUser.ModifyTime = tn
	} else if users[0].Type == authcommon.OwnerUserRole {
		mainUser = users[0]
		mainUser.CreateTime = tn
		mainUser.ModifyTime = tn
	} else {
		return errors.New("invalid init users info, must be have main user info")
	}

	if err := m.handler.Execute(true, func(tx *bolt.Tx) error {
		saveFunc := func(user *authcommon.User, rule *authcommon.StrategyDetail) error {
			rule.Owner = user.ID
			rule.Principals[0].PrincipalID = user.ID
			saveUser, err := m.getUser(tx, user.ID)
			if err != nil {
				return err
			}

			if saveUser == nil {
				// 添加主账户主体信息
				if err := saveValue(tx, tblUser, user.ID, converToUserStore(user)); err != nil {
					log.Error("[Store][User] save user fail", zap.Error(err), zap.String("name", user.Name))
					return err
				}
			}

			saveRule, err := m.getStrategyDetail(tx, rule.ID)
			if err != nil {
				return err
			}

			if saveRule == nil {
				// 添加主账户的默认鉴权策略信息
				if err := saveValue(tx, tblStrategy, rule.ID, convertForStrategyStore(rule)); err != nil {
					log.Error("[Store][Strategy] save auth_strategy", zap.Error(err),
						zap.String("name", rule.Name), zap.String("owner", rule.Owner))
					return err
				}
			}

			return nil
		}

		if superUser != nil {
			if err := saveFunc(superUser, superDefaultStrategy); err != nil {
				return err
			}
		}
		if err := saveFunc(mainUser, mainDefaultStrategy); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	tx, err := m.handle.StartTx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	// 挨个处理其他用户数据信息
	for i := 1; i < len(users); i++ {
		if err := m.AddUser(tx, users[i]); err != nil {
			return nil
		}
	}
	return tx.Commit()
}
