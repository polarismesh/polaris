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

package sqldb

import (
	"database/sql"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type configFileTemplateStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateConfigFileTemplate create config file template
func (cf *configFileTemplateStore) CreateConfigFileTemplate(
	template *model.ConfigFileTemplate) (*model.ConfigFileTemplate, error) {
	createSql := `
	INSERT INTO config_file_template (name, content, comment, format, create_time
		, create_by, modify_time, modify_by)
	VALUES (?, ?, ?, ?, sysdate()
		, ?, sysdate(), ?)
	`
	_, err := cf.master.Exec(createSql, template.Name, template.Content, template.Comment, template.Format,
		template.CreateBy, template.ModifyBy)
	if err != nil {
		return nil, store.Error(err)
	}

	return cf.GetConfigFileTemplate(template.Name)
}

// GetConfigFileTemplate get config file template by name
func (cf *configFileTemplateStore) GetConfigFileTemplate(name string) (*model.ConfigFileTemplate, error) {
	querySql := cf.baseSelectConfigFileTemplateSql() + " WHERE name = ?"
	rows, err := cf.master.Query(querySql, name)
	if err != nil {
		return nil, store.Error(err)
	}

	templates, err := cf.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(templates) > 0 {
		return templates[0], nil
	}
	return nil, nil
}

// QueryAllConfigFileTemplates query all config file templates
func (cf *configFileTemplateStore) QueryAllConfigFileTemplates() ([]*model.ConfigFileTemplate, error) {
	querySql := cf.baseSelectConfigFileTemplateSql() + " ORDER BY id DESC"
	rows, err := cf.master.Query(querySql)
	if err != nil {
		return nil, store.Error(err)
	}

	templates, err := cf.transferRows(rows)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (cf *configFileTemplateStore) baseSelectConfigFileTemplateSql() string {
	return `
SELECT id, name, content
	, IFNULL(comment, ''), format
	, UNIX_TIMESTAMP(create_time)
	, IFNULL(create_by, '')
	, UNIX_TIMESTAMP(modify_time)
	, IFNULL(modify_by, '')
FROM config_file_template 
	`
}

func (cf *configFileTemplateStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileTemplate, error) {
	if rows == nil {
		return nil, nil
	}
	defer func() {
		_ = rows.Close()
	}()

	var templates []*model.ConfigFileTemplate
	for rows.Next() {
		template := &model.ConfigFileTemplate{}
		var ctime, mtime int64
		err := rows.Scan(&template.Id, &template.Name, &template.Content, &template.Comment, &template.Format,
			&ctime, &template.CreateBy, &mtime, &template.ModifyBy)
		if err != nil {
			return nil, err
		}
		template.CreateTime = time.Unix(ctime, 0)
		template.ModifyTime = time.Unix(mtime, 0)

		templates = append(templates, template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}
