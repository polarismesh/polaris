/*
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

package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/namespace"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"

	_ "github.com/go-sql-driver/mysql"
	"github.com/polarismesh/polaris-server/store/boltdb"
	"github.com/polarismesh/polaris-server/store/sqldb"

	_ "github.com/polarismesh/polaris-server/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/cache"
	_ "github.com/polarismesh/polaris-server/store/boltdb"
	_ "github.com/polarismesh/polaris-server/store/sqldb"

	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/plugin/cmdb/memory"

	_ "github.com/polarismesh/polaris-server/plugin/auth/platform"
	_ "github.com/polarismesh/polaris-server/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris-server/plugin/discoverstat/discoverlocal"
	_ "github.com/polarismesh/polaris-server/plugin/history/logger"
	_ "github.com/polarismesh/polaris-server/plugin/password"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/lrurate"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris-server/plugin/statis/local"

	_ "github.com/polarismesh/polaris-server/plugin/healthchecker/heartbeatmemory"
	_ "github.com/polarismesh/polaris-server/plugin/healthchecker/heartbeatredis"
)

type TestConfig struct {
	Cache     cache.Config     `yaml:"cache"`
	Namespace namespace.Config `yaml:"namespace"`
	Config    Config           `yaml:"config"`
	Store     store.Config     `yaml:"store"`
	Auth      auth.Config      `yaml:"auth"`
	Plugin    plugin.Config    `yaml:"plugin"`
}

type ConfigCenterTest struct {
	cfg         *TestConfig
	once        sync.Once
	testService ConfigCenterServer
	testServer  *Server
	defaultCtx  context.Context
	storage     store.Store
}

func newConfigCenterTest(t *testing.T) (*ConfigCenterTest, error) {
	if err := os.RemoveAll("./config_center_test.bolt"); err != nil {
		return nil, err
	}

	c := &ConfigCenterTest{
		defaultCtx: context.Background(),
		testServer: new(Server),
		once:       sync.Once{},
		cfg:        new(TestConfig),
	}

	if err := c.doInitialize(); err != nil {
		fmt.Printf("bootstrap config test module error. %s", err.Error())
		return nil, err
	}

	return c, nil
}

func (c *ConfigCenterTest) doInitialize() error {
	var err error

	c.once.Do(func() {
		logOptions := log.DefaultOptions()
		_ = log.Configure(logOptions)
		// 加载启动配置文件
		err = c.loadBootstrapConfig()
		if err != nil {
			return
		}

		// 初始化defaultCtx
		c.defaultCtx = context.WithValue(c.defaultCtx, utils.StringContext("request-id"), "config-test-request-id")
		c.defaultCtx = context.WithValue(c.defaultCtx, utils.ContextUserNameKey, "polaris")
		c.defaultCtx = context.WithValue(c.defaultCtx, utils.ContextAuthTokenKey, "4azbewS+pdXvrMG1PtYV3SrcLxjmYd0IVNaX9oYziQygRnKzjcSbxl+Reg7zYQC1gRrGiLzmMY+w+aCxOYI=")

		plugin.SetPluginConfig(&c.cfg.Plugin)

		// 初始化存储层
		store.SetStoreConfig(&c.cfg.Store)
		s, err := store.GetStore()
		if err != nil {
			fmt.Printf("[ERROR] configure get store fail: %v\n", err)
			return
		}
		c.storage = s

		if err := cache.TestCacheInitialize(context.Background(), &c.cfg.Cache, s); err != nil {
			fmt.Printf("[ERROR] configure init cache fail: %v\n", err)
			return
		}

		cacheMgr, err := cache.GetCacheManager()
		if err != nil {
			fmt.Printf("[ERROR] configure get cache fail: %v\n", err)
			return
		}

		if err := auth.Initialize(context.Background(), &c.cfg.Auth, s, cacheMgr); err != nil {
			fmt.Printf("[ERROR] configure init auth fail: %v\n", err)
			return
		}

		authSvr, err := auth.GetAuthServer()
		if err != nil {
			fmt.Printf("[ERROR] configure get auth fail: %v\n", err)
			return
		}

		// 初始化配置中心模块
		if err := c.testServer.initialize(context.Background(), c.cfg.Config, s, cacheMgr, authSvr); err != nil {
			return
		}

		c.testService = newServerAuthAbility(c.testServer, authSvr)
	})
	return err
}

func (c *ConfigCenterTest) loadBootstrapConfig() error {
	file, err := os.Open("test.yaml")
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	err = yaml.NewDecoder(file).Decode(&c.cfg)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	return err
}

func (c *ConfigCenterTest) clearTestData() error {
	c.testServer.Cache().Clear()

	if c.storage.Name() == sqldb.STORENAME {
		if err := c.clearTestDataWhenUseRDS(); err != nil {
			return err
		}
	} else if c.storage.Name() == boltdb.STORENAME {
		if err := c.clearTestDataWhenUseBoltdb(); err != nil {
			return err
		}
	} else {
		return errors.New("store impl unexpect")
	}

	return nil
}

func (c *ConfigCenterTest) clearTestDataWhenUseBoltdb() error {

	proxyTx, err := c.storage.StartTx()
	if err != nil {
		return err
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)

	bucketName := []string{
		"ConfigFileGroup",
		"ConfigFileGroupID",
		"ConfigFile",
		"ConfigFileID",
		"ConfigFileReleaseHistory",
		"ConfigFileReleaseHistoryID",
		"ConfigFileRelease",
		"ConfigFileReleaseID",
		"ConfigFileTag",
		"ConfigFileTagID",
		"namespace",
	}

	defer tx.Rollback()

	for i := range bucketName {
		if err := tx.DeleteBucket([]byte(bucketName[i])); err != nil {
			if !errors.Is(err, bolt.ErrBucketNotFound) {
				return err
			}
		}
	}

	return tx.Commit()
}

func (c *ConfigCenterTest) clearTestDataWhenUseRDS() error {

	proxyTx, err := c.storage.StartTx()
	if err != nil {
		return err
	}

	tx := proxyTx.GetDelegateTx().(*sql.Tx)

	_, err = tx.Exec("delete from config_file_group where namespace = ? ", testNamespace)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from config_file where namespace = ? ", testNamespace)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from config_file_release where namespace = ? ", testNamespace)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from config_file_release_history where namespace = ? ", testNamespace)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from config_file_tag where namespace = ? ", testNamespace)
	if err != nil {
		return err
	}
	_, err = tx.Exec("delete from namespace where name = ? ", testNamespace)
	if err != nil {
		return err
	}

	// 清理缓存
	originServer.Cache().Clear()

	return err
}

func randomStr() string {
	uuid, _ := uuid.NewUUID()
	return uuid.String()
}

func assembleConfigFileGroup() *api.ConfigFileGroup {
	return &api.ConfigFileGroup{
		Namespace: utils.NewStringValue(testNamespace),
		Name:      utils.NewStringValue(testGroup),
		Comment:   utils.NewStringValue("autotest"),
	}
}

func assembleRandomConfigFileGroup() *api.ConfigFileGroup {
	return &api.ConfigFileGroup{
		Namespace: utils.NewStringValue(testNamespace),
		Name:      utils.NewStringValue(randomGroupPrefix + randomStr()),
		Comment:   utils.NewStringValue("autotest"),
	}
}

func assembleConfigFile() *api.ConfigFile {
	tag1 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v1"),
	}

	tag2 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v2"),
	}

	tag3 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k2"),
		Value: utils.NewStringValue("v1"),
	}

	return &api.ConfigFile{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(testGroup),
		Name:      utils.NewStringValue(testFile),
		Format:    utils.NewStringValue(utils.FileFormatText),
		Content:   utils.NewStringValue("k1=v1,k2=v2"),
		Tags:      []*api.ConfigFileTag{tag1, tag2, tag3},
		CreateBy:  utils.NewStringValue(operator),
	}
}

func assembleConfigFileWithFixedGroupAndRandomFileName(group string) *api.ConfigFile {
	tag1 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v1"),
	}

	tag2 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v2"),
	}

	tag3 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k2"),
		Value: utils.NewStringValue("v1"),
	}

	return &api.ConfigFile{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(group),
		Name:      utils.NewStringValue(randomStr()),
		Format:    utils.NewStringValue(utils.FileFormatText),
		Content:   utils.NewStringValue("k1=v1,k2=v2"),
		Tags:      []*api.ConfigFileTag{tag1, tag2, tag3},
		CreateBy:  utils.NewStringValue(operator),
	}
}

func assembleConfigFileWithRandomGroupAndFixedFileName(fileName string) *api.ConfigFile {
	tag1 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v1"),
	}

	tag2 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v2"),
	}

	tag3 := &api.ConfigFileTag{
		Key:   utils.NewStringValue("k2"),
		Value: utils.NewStringValue("v1"),
	}

	return &api.ConfigFile{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(randomStr()),
		Name:      utils.NewStringValue(fileName),
		Format:    utils.NewStringValue(utils.FileFormatText),
		Content:   utils.NewStringValue("k1=v1,k2=v2"),
		Tags:      []*api.ConfigFileTag{tag1, tag2, tag3},
		CreateBy:  utils.NewStringValue(operator),
	}
}

func assembleConfigFileRelease(configFile *api.ConfigFile) *api.ConfigFileRelease {
	return &api.ConfigFileRelease{
		Name:      utils.NewStringValue("release-name"),
		Namespace: configFile.Namespace,
		Group:     configFile.Group,
		FileName:  configFile.Name,
		CreateBy:  utils.NewStringValue("polaris"),
	}
}

func assembleDefaultClientConfigFile(version uint64) []*api.ClientConfigFileInfo {
	return []*api.ClientConfigFileInfo{
		{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			FileName:  utils.NewStringValue(testFile),
			Version:   utils.NewUInt64Value(version),
		},
	}
}
