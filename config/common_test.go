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

package config_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	_ "github.com/polarismesh/polaris/plugin/crypto/aes"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/memory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/redis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	"github.com/polarismesh/polaris/store"
	_ "github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/mysql"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap Bootstrap        `yaml:"bootstrap"`
	Cache     cache.Config     `yaml:"cache"`
	Namespace namespace.Config `yaml:"namespace"`
	Config    config.Config    `yaml:"config"`
	Store     store.Config     `yaml:"store"`
	Auth      auth.Config      `yaml:"auth"`
	Plugin    plugin.Config    `yaml:"plugin"`
}

type ConfigCenterTest struct {
	testsuit.DiscoverTestSuit
	cfg *TestConfig
}

func (c *ConfigCenterTest) clearTestData() error {
	defer func() {
		time.Sleep(5 * time.Second)
	}()
	return c.GetTestDataClean().ClearTestDataWhenUseRDS()
}

func randomStr() string {
	uuid, _ := uuid.NewUUID()
	return uuid.String()
}

func assembleConfigFileGroup() *apiconfig.ConfigFileGroup {
	return &apiconfig.ConfigFileGroup{
		Namespace: utils.NewStringValue(testNamespace),
		Name:      utils.NewStringValue(testGroup),
		Comment:   utils.NewStringValue("autotest"),
	}
}

func assembleRandomConfigFileGroup() *apiconfig.ConfigFileGroup {
	return &apiconfig.ConfigFileGroup{
		Namespace: utils.NewStringValue(testNamespace),
		Name:      utils.NewStringValue(randomGroupPrefix + randomStr()),
		Comment:   utils.NewStringValue("autotest"),
	}
}

func assembleConfigFile() *apiconfig.ConfigFile {
	tag1 := &apiconfig.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v1"),
	}
	tag2 := &apiconfig.ConfigFileTag{
		Key:   utils.NewStringValue("k1"),
		Value: utils.NewStringValue("v2"),
	}
	tag3 := &apiconfig.ConfigFileTag{
		Key:   utils.NewStringValue("k2"),
		Value: utils.NewStringValue("v1"),
	}
	return &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(testGroup),
		Name:      utils.NewStringValue(testFile),
		Format:    utils.NewStringValue(utils.FileFormatText),
		Content:   utils.NewStringValue("k1=v1,k2=v2"),
		Tags:      []*apiconfig.ConfigFileTag{tag1, tag2, tag3},
		CreateBy:  utils.NewStringValue(operator),
	}
}

func assembleEncryptConfigFile() *apiconfig.ConfigFile {
	configFile := assembleConfigFile()
	configFile.Encrypted = utils.NewBoolValue(true)
	configFile.EncryptAlgo = utils.NewStringValue("AES")
	return configFile
}

func assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name string) *apiconfig.ConfigFile {
	configFile := assembleConfigFile()
	configFile.Namespace = utils.NewStringValue(namespace)
	configFile.Group = utils.NewStringValue(group)
	configFile.Name = utils.NewStringValue(name)
	return configFile
}

func assembleConfigFileWithFixedGroupAndRandomFileName(group string) *apiconfig.ConfigFile {
	configFile := assembleConfigFile()
	configFile.Group = utils.NewStringValue(group)
	configFile.Name = utils.NewStringValue(randomStr())
	return configFile
}

func assembleConfigFileWithRandomGroupAndFixedFileName(name string) *apiconfig.ConfigFile {
	configFile := assembleConfigFile()
	configFile.Group = utils.NewStringValue(randomStr())
	configFile.Name = utils.NewStringValue(name)
	return configFile
}

func assembleConfigFileRelease(configFile *apiconfig.ConfigFile) *apiconfig.ConfigFileRelease {
	return &apiconfig.ConfigFileRelease{
		Name:      utils.NewStringValue("release-name-" + uuid.NewString()),
		Namespace: configFile.Namespace,
		Group:     configFile.Group,
		FileName:  configFile.Name,
		CreateBy:  utils.NewStringValue("polaris"),
	}
}

func assembleDefaultClientConfigFile(version uint64) []*apiconfig.ClientConfigFileInfo {
	return []*apiconfig.ClientConfigFileInfo{
		{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			FileName:  utils.NewStringValue(testFile),
			Version:   utils.NewUInt64Value(version),
		},
	}
}

func newConfigCenterTestSuit(t *testing.T) *ConfigCenterTest {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})
	return testSuit
}
