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
	"database/sql"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"github.com/polarismesh/polaris-server/store/boltdb"
	"github.com/polarismesh/polaris-server/store/sqldb"
)

func clearTestData() error {
	originServer.Cache().Clear()

	s, err := store.GetStore()
	if err != nil {
		return err
	}

	if s.Name() == sqldb.STORENAME {
		if err := clearTestDataWhenUseRDS(); err != nil {
			return err
		}
	} else if s.Name() == boltdb.STORENAME {
		if err := clearTestDataWhenUseBoltdb(); err != nil {
			return err
		}
	} else {
		return errors.New("store impl unexpect")
	}

	return nil
}

func clearTestDataWhenUseBoltdb() error {
	s, err := store.GetStore()
	if err != nil {
		return err
	}

	proxyTx, err := s.StartTx()
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

func clearTestDataWhenUseRDS() error {

	s, err := store.GetStore()
	if err != nil {
		return err
	}

	proxyTx, err := s.StartTx()
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
