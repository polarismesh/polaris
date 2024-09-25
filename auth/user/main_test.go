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

package defaultuser_test

import (
	"errors"

	_ "github.com/go-sql-driver/mysql"
	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	_ "github.com/polarismesh/polaris/plugin/cmdb/memory"
	_ "github.com/polarismesh/polaris/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/memory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/redis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris/plugin/statis/logger"
	_ "github.com/polarismesh/polaris/plugin/statis/prometheus"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/mysql"
	sqldb "github.com/polarismesh/polaris/store/mysql"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

const (
	tblUser     string = "user"
	tblStrategy string = "strategy"
	tblGroup    string = "group"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap    Bootstrap          `yaml:"bootstrap"`
	Cache        cache.Config       `yaml:"cache"`
	Namespace    namespace.Config   `yaml:"namespace"`
	HealthChecks healthcheck.Config `yaml:"healthcheck"`
	Store        store.Config       `yaml:"store"`
	Auth         auth.Config        `yaml:"auth"`
	Plugin       plugin.Config      `yaml:"plugin"`
}

type AuthTestSuit struct {
	testsuit.DiscoverTestSuit
}

// 判断一个resp是否执行成功
func respSuccess(resp api.ResponseMessage) bool {

	ret := api.CalcCode(resp) == 200

	return ret
}

type options func(cfg *TestConfig)

func (d *AuthTestSuit) cleanAllUser() {
	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from user where name like 'test%'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblUser)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

func (d *AuthTestSuit) cleanAllUserGroup() {
	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from user_group where name like 'test%'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from user_group_relation"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblGroup)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

func (d *AuthTestSuit) cleanAllAuthStrategy() {
	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from auth_strategy where id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from auth_principal where strategy_id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from auth_strategy_resource where strategy_id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblStrategy)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}
