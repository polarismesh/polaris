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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type configFileGroupStore struct {
	db *BaseDB
}

// CreateConfigFileGroup 创建配置文件组
func (fg *configFileGroupStore) CreateConfigFileGroup(
	fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error) {

	createSql := "insert into config_file_group(name, namespace,comment,create_time, create_by, " +
		" modify_time, modify_by, owner)" +
		"value (?,?,?,sysdate(),?,sysdate(),?,?)"
	_, err := fg.db.Exec(createSql, fileGroup.Name, fileGroup.Namespace, fileGroup.Comment,
		fileGroup.CreateBy, fileGroup.ModifyBy, fileGroup.Owner)
	if err != nil {
		return nil, store.Error(err)
	}

	return fg.GetConfigFileGroup(fileGroup.Namespace, fileGroup.Name)
}

// GetConfigFileGroup 获取配置文件组
func (fg *configFileGroupStore) GetConfigFileGroup(namespace, name string) (*model.ConfigFileGroup, error) {
	querySql := fg.genConfigFileGroupSelectSql() + " where namespace=? and name=?"
	rows, err := fg.db.Query(querySql, namespace, name)
	if err != nil {
		return nil, store.Error(err)
	}
	cfgs, err := fg.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(cfgs) > 0 {
		return cfgs[0], nil
	}
	return nil, nil
}

// QueryConfigFileGroups 翻页查询配置文件组, name 为模糊匹配关键字
func (fg *configFileGroupStore) QueryConfigFileGroups(namespace, name string,
	offset, limit uint32) (uint32, []*model.ConfigFileGroup, error) {

	name = "%" + name + "%"

	// 全部 namespace
	if namespace == "" {
		countSql := "select count(*) from config_file_group where name like ?"
		var count uint32
		err := fg.db.QueryRow(countSql, name).Scan(&count)
		if err != nil {
			return count, nil, err
		}

		sql := fg.genConfigFileGroupSelectSql() + " where name like ? order by id desc limit ?,?"
		rows, err := fg.db.Query(sql, name, offset, limit)
		if err != nil {
			return 0, nil, err
		}
		cfgs, err := fg.transferRows(rows)
		if err != nil {
			return 0, nil, err
		}

		return count, cfgs, nil
	}

	// 特定 namespace
	countSql := "select count(*) from config_file_group where namespace=? and name like ?"
	var count uint32
	err := fg.db.QueryRow(countSql, namespace, name).Scan(&count)
	if err != nil {
		return count, nil, err
	}

	sql := fg.genConfigFileGroupSelectSql() + " where namespace=? and name like ? order by id desc limit ?,? "
	rows, err := fg.db.Query(sql, namespace, name, offset, limit)
	if err != nil {
		return 0, nil, err
	}
	cfgs, err := fg.transferRows(rows)
	if err != nil {
		return 0, nil, err
	}

	return count, cfgs, nil
}

// DeleteConfigFileGroup 删除配置文件组
func (fg *configFileGroupStore) DeleteConfigFileGroup(namespace, name string) error {
	deleteSql := "delete from config_file_group where namespace = ? and name=?"

	log.Infof("[Config][Storage] delete config file group(%s, %s)", namespace, name)
	if _, err := fg.db.Exec(deleteSql, namespace, name); err != nil {
		return err
	}

	return nil
}

// UpdateConfigFileGroup 更新配置文件组信息
func (fg *configFileGroupStore) UpdateConfigFileGroup(
	fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error) {

	updateSql := "update config_file_group set comment = ?, modify_time = sysdate(), modify_by = ? " +
		" where namespace = ? and name = ?"
	_, err := fg.db.Exec(updateSql, fileGroup.Comment, fileGroup.ModifyBy, fileGroup.Namespace, fileGroup.Name)
	if err != nil {
		return nil, store.Error(err)
	}
	return fg.GetConfigFileGroup(fileGroup.Namespace, fileGroup.Name)
}

// FindConfigFileGroups 获取一组配置文件组信息
func (fg *configFileGroupStore) FindConfigFileGroups(namespace string,
	names []string) ([]*model.ConfigFileGroup, error) {

	querySql := fg.genConfigFileGroupSelectSql()
	params := make([]interface{}, 0)

	if namespace == "" {
		querySql += " where name in (%s)"
	} else {
		querySql += " where namespace = ? and name in (%s)"
		params = append(params, namespace)
	}

	inParamPlaceholders := make([]string, 0)
	for i := 0; i < len(names); i++ {
		inParamPlaceholders = append(inParamPlaceholders, "?")
		params = append(params, names[i])
	}
	querySql = fmt.Sprintf(querySql, strings.Join(inParamPlaceholders, ","))

	rows, err := fg.db.Query(querySql, params...)
	if err != nil {
		return nil, err
	}
	cfgs, err := fg.transferRows(rows)
	if err != nil {
		return nil, err
	}
	return cfgs, nil
}

func (fg *configFileGroupStore) GetConfigFileGroupById(id uint64) (*model.ConfigFileGroup, error) {
	querySql := fg.genConfigFileGroupSelectSql()
	querySql += fmt.Sprintf(" where id = %s", strconv.FormatUint(id, 10))

	rows, err := fg.db.Query(querySql)
	if err != nil {
		return nil, err
	}

	cfgs, err := fg.transferRows(rows)
	if err != nil {
		return nil, err
	}

	return cfgs[0], nil
}

func (fg *configFileGroupStore) genConfigFileGroupSelectSql() string {
	return "select id,name,namespace,IFNULL(comment,''),UNIX_TIMESTAMP(create_time),IFNULL(create_by,'')," +
		"UNIX_TIMESTAMP(modify_time),IFNULL(modify_by,''),IFNULL(owner,'') from config_file_group"
}

func (fg *configFileGroupStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileGroup, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var fileGroups []*model.ConfigFileGroup

	for rows.Next() {
		fileGroup := &model.ConfigFileGroup{}
		var ctime, mtime int64
		err := rows.Scan(&fileGroup.Id, &fileGroup.Name, &fileGroup.Namespace, &fileGroup.Comment, &ctime,
			&fileGroup.CreateBy, &mtime, &fileGroup.ModifyBy, &fileGroup.Owner)
		if err != nil {
			return nil, err
		}
		fileGroup.CreateTime = time.Unix(ctime, 0)
		fileGroup.ModifyTime = time.Unix(mtime, 0)

		fileGroups = append(fileGroups, fileGroup)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return fileGroups, nil
}
