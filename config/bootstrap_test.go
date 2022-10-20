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
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/auth"
	_ "github.com/polarismesh/polaris/auth/defaultauth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	_ "github.com/polarismesh/polaris/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/heartbeatmemory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/heartbeatredis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/boltdb"
	"github.com/polarismesh/polaris/store/sqldb"
	_ "github.com/polarismesh/polaris/store/sqldb"
	"github.com/polarismesh/polaris/testdata"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap Bootstrap        `yaml:"bootstrap"`
	Cache     cache.Config     `yaml:"cache"`
	Namespace namespace.Config `yaml:"namespace"`
	Config    Config           `yaml:"config"`
	Store     store.Config     `yaml:"store"`
	Auth      auth.Config      `yaml:"auth"`
	Plugin    plugin.Config    `yaml:"plugin"`
}

type ConfigCenterTest struct {
	cfg         *TestConfig
	testService ConfigCenterServer
	testServer  *Server
	defaultCtx  context.Context
	cancel      context.CancelFunc
	storage     store.Store
}

func newConfigCenterTest(t *testing.T) (*ConfigCenterTest, error) {
	if err := os.RemoveAll("./config_center_test.bolt"); err != nil {
		return nil, err
	}

	c := &ConfigCenterTest{
		defaultCtx: context.Background(),
		testServer: new(Server),
		cfg:        new(TestConfig),
	}

	if err := c.doInitialize(); err != nil {
		fmt.Printf("bootstrap config test module error. %s", err.Error())
		return nil, err
	}

	return c, nil
}

func (c *ConfigCenterTest) doInitialize() error {
	// 加载启动配置文件
	if err := c.loadBootstrapConfig(); err != nil {
		return err
	}
	_ = commonlog.Configure(c.cfg.Bootstrap.Logger)
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	plugin.SetPluginConfig(&c.cfg.Plugin)

	// 初始化存储层
	store.SetStoreConfig(&c.cfg.Store)
	s, err := store.TestGetStore()
	if err != nil {
		fmt.Printf("[ERROR] configure get store fail: %v\n", err)
		return err
	}
	c.storage = s

	cacheMgr, err := cache.TestCacheInitialize(ctx, &c.cfg.Cache, s)
	if err != nil {
		fmt.Printf("[ERROR] configure init cache fail: %v\n", err)
		return err
	}

	authSvr, err := auth.TestInitialize(ctx, &c.cfg.Auth, s, cacheMgr)
	if err != nil {
		fmt.Printf("[ERROR] configure init auth fail: %v\n", err)
		return err
	}

	nsOp, err := namespace.TestInitialize(ctx, &c.cfg.Namespace, s, cacheMgr, authSvr)
	if err != nil {
		fmt.Printf("[ERROR] configure init cache fail: %v\n", err)
		return err
	}

	// 初始化配置中心模块
	if err := c.testServer.initialize(ctx, c.cfg.Config, s, nsOp, cacheMgr, authSvr); err != nil {
		return err
	}
	c.testServer.initialized = true
	c.testService = newServerAuthAbility(c.testServer, authSvr)

	time.Sleep(5 * time.Second)

	return nil
}

func (c *ConfigCenterTest) loadBootstrapConfig() error {
	confFileName := testdata.Path("config_test.yaml")

	// 初始化defaultCtx
	c.defaultCtx = context.WithValue(c.defaultCtx, utils.StringContext("request-id"), "config-test-request-id")
	c.defaultCtx = context.WithValue(c.defaultCtx, utils.ContextUserNameKey, "polaris")

	if os.Getenv("STORE_MODE") == "sqldb" {
		fmt.Printf("run store mode : sqldb\n")
		confFileName = testdata.Path("config_test_sqldb.yaml")
		c.defaultCtx = context.WithValue(c.defaultCtx, utils.ContextAuthTokenKey, "nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=")
	} else {
		c.defaultCtx = context.WithValue(c.defaultCtx, utils.ContextAuthTokenKey, "nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=")
	}

	file, err := os.Open(confFileName)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	err = yaml.NewDecoder(file).Decode(c.cfg)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	return err
}

func (c *ConfigCenterTest) clearTestData() error {
	defer func() {
		c.cancel()
		time.Sleep(5 * time.Second)

		c.storage.Destroy()
		time.Sleep(5 * time.Second)
	}()

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

	tx := proxyTx.GetDelegateTx().(*sqldb.BaseTx)

	defer tx.Rollback()

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
	c.testServer.Cache().CleanAll()

	return tx.Commit()
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
