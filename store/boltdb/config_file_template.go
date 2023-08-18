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

package boltdb

import (
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	tblConfigFileTemplate   string = "ConfigFileTemplate"
	tblConfigFileTemplateID string = "ConfigFileTemplateID"
)

type configFileTemplateStore struct {
	handler BoltHandler
}

func newConfigFileTemplateStore(handler BoltHandler) *configFileTemplateStore {
	s := &configFileTemplateStore{handler: handler}
	return s
}

// QueryAllConfigFileTemplates query all config file templates
func (cf *configFileTemplateStore) QueryAllConfigFileTemplates() ([]*model.ConfigFileTemplate, error) {
	ret, err := cf.handler.LoadValuesAll(tblConfigFileTemplate, &model.ConfigFileTemplate{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	var templates []*model.ConfigFileTemplate
	for _, v := range ret {
		templates = append(templates, v.(*model.ConfigFileTemplate))
	}
	return templates, nil
}

// GetConfigFileTemplate get config file template
func (cf *configFileTemplateStore) GetConfigFileTemplate(name string) (*model.ConfigFileTemplate, error) {
	proxy, err := cf.handler.StartTx()
	if err != nil {
		return nil, err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer func() {
		_ = tx.Rollback()
	}()

	values := make(map[string]interface{})
	if err = loadValues(tx, tblConfigFileTemplate, []string{name}, &model.ConfigFileTemplate{}, values); err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	if len(values) > 1 {
		return nil, ErrMultipleConfigFileFound
	}

	data := values[name].(*model.ConfigFileTemplate)

	return data, nil
}

// CreateConfigFileTemplate create config file template
func (cf *configFileTemplateStore) CreateConfigFileTemplate(
	template *model.ConfigFileTemplate) (*model.ConfigFileTemplate, error) {
	proxy, err := cf.handler.StartTx()
	if err != nil {
		return nil, err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer func() {
		_ = tx.Rollback()
	}()

	table, err := tx.CreateBucketIfNotExists([]byte(tblConfigFile))
	if err != nil {
		return nil, store.Error(err)
	}
	nextId, err := table.NextSequence()
	if err != nil {
		return nil, store.Error(err)
	}

	template.Id = nextId
	template.CreateTime = time.Now()
	template.ModifyTime = time.Now()

	key := template.Name
	if err := saveValue(tx, tblConfigFileTemplate, key, template); err != nil {
		log.Error("[ConfigFileTemplate] save error", zap.Error(err))
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		log.Error("[ConfigFileTemplate] commit error", zap.Error(err))
		return nil, err
	}

	return template, nil
}
