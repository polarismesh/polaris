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

package sqldb

import (
	"database/sql"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type configFileTagStore struct {
	db *BaseDB
}

// CreateConfigFileTag 创建配置文件标签
func (t *configFileTagStore) CreateConfigFileTag(tx store.Tx, fileTag *model.ConfigFileTag) error {
	insertSql := "insert into config_file_tag(`key`,`value`,namespace,`group`,file_name,create_time, " +
		" create_by,modify_time,modify_by)" +
		"values(?,?,?,?,?,sysdate(),?,sysdate(),?)"

	var err error
	if tx != nil {
		_, err = tx.GetDelegateTx().(*BaseTx).Exec(insertSql, fileTag.Key, fileTag.Value, fileTag.Namespace,
			fileTag.Group, fileTag.FileName, fileTag.CreateBy, fileTag.ModifyBy)
	} else {
		_, err = t.db.Exec(insertSql, fileTag.Key, fileTag.Value, fileTag.Namespace,
			fileTag.Group, fileTag.FileName, fileTag.CreateBy, fileTag.ModifyBy)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// QueryConfigFileByTag 通过标签查询配置文件
func (t *configFileTagStore) QueryConfigFileByTag(namespace, group, fileName string,
	tags ...string) ([]*model.ConfigFileTag, error) {

	group = "%" + group + "%"
	fileName = "%" + fileName + "%"
	querySql := t.baseSelectSql() + " where namespace = ? and `group` like ? and file_name like ? "

	var tagWhereSql []string
	for i := 0; i < len(tags)/2; i++ {
		tagWhereSql = append(tagWhereSql, "(?,?)")
	}
	tagIn := "and (`key`, `value`) in  (" + strings.Join(tagWhereSql, ",") + ")"
	querySql = querySql + tagIn

	params := []interface{}{namespace, group, fileName}
	for _, tag := range tags {
		params = append(params, tag)
	}
	rows, err := t.db.Query(querySql, params...)
	if err != nil {
		return nil, store.Error(err)
	}

	result, err := t.transferRows(rows)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// QueryTagByConfigFile 查询配置文件标签
func (t *configFileTagStore) QueryTagByConfigFile(namespace, group, fileName string) ([]*model.ConfigFileTag, error) {
	querySql := t.baseSelectSql() + " where namespace = ? and `group` = ? and file_name = ?"
	rows, err := t.db.Query(querySql, namespace, group, fileName)
	if err != nil {
		return nil, store.Error(err)
	}

	tags, err := t.transferRows(rows)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// DeleteConfigFileTag 删除配置文件标签
func (t *configFileTagStore) DeleteConfigFileTag(tx store.Tx, namespace, group, fileName, key, value string) error {
	deleteSql := "delete from config_file_tag where `key` = ? and `value` = ? and namespace = ? " +
		" and `group` = ? and file_name = ?"
	var err error
	if tx != nil {
		_, err = tx.GetDelegateTx().(*BaseTx).Exec(deleteSql, key, value, namespace, group, fileName)
	} else {
		_, err = t.db.Exec(deleteSql, key, value, namespace, group, fileName)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// DeleteTagByConfigFile 删除配置文件的标签
func (t *configFileTagStore) DeleteTagByConfigFile(tx store.Tx, namespace, group, fileName string) error {
	deleteSql := "delete from config_file_tag where namespace = ? and `group` = ? and file_name = ?"
	var err error
	if tx != nil {
		_, err = tx.GetDelegateTx().(*BaseTx).Exec(deleteSql, namespace, group, fileName)
	} else {
		_, err = t.db.Exec(deleteSql, namespace, group, fileName)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

func (t *configFileTagStore) baseSelectSql() string {
	return "select id, `key`,`value`,namespace,`group`,file_name,UNIX_TIMESTAMP(create_time), " +
		" IFNULL(create_by, ''),UNIX_TIMESTAMP(modify_time),IFNULL(modify_by, '') from config_file_tag"
}

func (t *configFileTagStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileTag, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var tags []*model.ConfigFileTag

	for rows.Next() {
		tag := &model.ConfigFileTag{}
		var ctime, mtime int64
		err := rows.Scan(&tag.Id, &tag.Key, &tag.Value, &tag.Namespace, &tag.Group, &tag.FileName,
			&ctime, &tag.CreateBy, &mtime, &tag.ModifyBy)
		if err != nil {
			return nil, err
		}
		tag.CreateTime = time.Unix(ctime, 0)
		tag.ModifyTime = time.Unix(mtime, 0)

		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
