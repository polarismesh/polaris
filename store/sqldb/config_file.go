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
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"time"
)

type configFileStore struct {
	db *BaseDB
}

func (cf *configFileStore) StartTx() (*sql.Tx, error) {
	tx, err := cf.db.Begin()
	return tx.Tx, err
}

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFile(tx *sql.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	err := cf.hardDeleteConfigFile(file.Namespace, file.Group, file.Name)
	if err != nil {
		return nil, err
	}

	createSql := "insert into config_file(name,namespace,`group`,content,comment,format,create_time,create_by,modify_time,modify_by) values " +
		"(?,?,?,?,?,?,sysdate(),?,sysdate(),?)"
	if tx != nil {
		_, err = tx.Exec(createSql, file.Name, file.Namespace, file.Group, file.Content, file.Comment, file.Format, file.CreateBy, file.ModifyBy)
	} else {
		_, err = cf.db.Exec(createSql, file.Name, file.Namespace, file.Group, file.Content, file.Comment, file.Format, file.CreateBy, file.ModifyBy)
	}
	if err != nil {
		return nil, store.Error(err)
	}

	return cf.GetConfigFile(tx, file.Namespace, file.Group, file.Name)
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) GetConfigFile(tx *sql.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	querySql := cf.baseSelectConfigFileSql() + "where namespace = ? and `group` = ? and name = ? and flag = 0"
	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(querySql, namespace, group, name)
	} else {
		rows, err = cf.db.Query(querySql, namespace, group, name)
	}
	if err != nil {
		return nil, err
	}
	files, err := cf.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(files) > 0 {
		return files[0], nil
	}

	return nil, nil
}

// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
func (cf *configFileStore) QueryConfigFiles(namespace, group, name string, offset, limit int) (uint32, []*model.ConfigFile, error) {
	group = "%" + group + "%"
	name = "%" + name + "%"
	countSql := "select count(*) from config_file where namespace = ? and `group` like ? and name like ? and flag = 0"

	var count uint32
	err := cf.db.QueryRow(countSql, namespace, group, name).Scan(&count)
	if err != nil {
		return 0, nil, err
	}

	querySql := cf.baseSelectConfigFileSql() + "where namespace = ? and `group` like ? and name like ? and flag = 0 limit ?,?"
	rows, err := cf.db.Query(querySql, namespace, group, name, offset, limit)
	if err != nil {
		return 0, nil, err
	}

	files, err := cf.transferRows(rows)
	if err != nil {
		return 0, nil, err
	}

	return count, files, nil

}

// UpdateConfigFile 更新配置文件
func (cf *configFileStore) UpdateConfigFile(tx *sql.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	updateSql := "update config_file set content = ? , comment = ?, format = ?, modify_time = sysdate(), modify_by = ? where namespace = ? and `group` = ? and name = ?"
	var err error
	if tx != nil {
		_, err = tx.Exec(updateSql, file.Content, file.Comment, file.Format, file.ModifyBy, file.Namespace, file.Group, file.Name)
	} else {
		_, err = cf.db.Exec(updateSql, file.Content, file.Comment, file.Format, file.ModifyBy, file.Namespace, file.Group, file.Name)
	}
	if err != nil {
		return nil, store.Error(err)
	}
	return cf.GetConfigFile(tx, file.Namespace, file.Group, file.Name)
}

// DeleteConfigFile 删除配置文件
func (cf *configFileStore) DeleteConfigFile(tx *sql.Tx, namespace, group, name string) error {
	deleteSql := "update config_file set flag = 1 where namespace = ? and `group` = ? and name = ?"
	var err error
	if tx != nil {
		_, err = tx.Exec(deleteSql, namespace, group, name)
	} else {
		_, err = cf.db.Exec(deleteSql, namespace, group, name)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

func (cf *configFileStore) baseSelectConfigFileSql() string {
	return "select id, name,namespace,`group`,content,IFNULL(comment, ''),format, UNIX_TIMESTAMP(create_time),IFNULL(create_by, ''),UNIX_TIMESTAMP(modify_time),IFNULL(modify_by, '') from config_file "
}

func (cf *configFileStore) hardDeleteConfigFile(namespace, group, name string) error {
	log.GetConfigLogger().Infof("[Config][Storage] delete config file. namespace = %s, group = %s, name = %s", namespace, group, name)

	deleteSql := "delete from config_file where namespace = ? and `group` = ? and name = ? and flag = 1"

	_, err := cf.db.Exec(deleteSql, namespace, group, name)
	if err != nil {
		return store.Error(err)
	}

	return nil
}

func (cf *configFileStore) transferRows(rows *sql.Rows) ([]*model.ConfigFile, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var files []*model.ConfigFile

	for rows.Next() {
		file := &model.ConfigFile{}
		var ctime, mtime int64
		err := rows.Scan(&file.Id, &file.Name, &file.Namespace, &file.Group, &file.Content, &file.Comment, &file.Format, &ctime, &file.CreateBy, &mtime, &file.ModifyBy)
		if err != nil {
			return nil, err
		}
		file.CreateTime = time.Unix(ctime, 0)
		file.ModifyTime = time.Unix(mtime, 0)

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}
