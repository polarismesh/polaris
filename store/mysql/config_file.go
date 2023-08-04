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
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var _ store.ConfigFileStore = (*configFileStore)(nil)

type configFileStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFileTx(tx store.Tx, file *model.ConfigFile) error {
	dbTx := tx.GetDelegateTx().(*BaseTx)
	deleteSql := "delete from config_file where namespace = ? and `group` = ? and name = ? and flag = 1"
	if _, err := dbTx.Exec(deleteSql, file.Namespace, file.Group, file.Name); err != nil {
		return store.Error(err)
	}

	createSql := "insert into config_file(name,namespace,`group`,content,comment,format,create_time, " +
		"create_by,modify_time,modify_by) values " +
		"(?,?,?,?,?,?,sysdate(),?,sysdate(),?)"
	if _, err := dbTx.Exec(createSql, file.Name, file.Namespace, file.Group,
		file.Content, file.Comment, file.Format, file.CreateBy, file.ModifyBy); err != nil {
		return store.Error(err)
	}

	if err := cf.batchCleanTags(dbTx, file); err != nil {
		return store.Error(err)
	}

	return nil
}

func (cf *configFileStore) batchAddTags(tx *BaseTx, file *model.ConfigFile) error {
	// 添加配置标签
	insertSql := "insert into config_file_tag(`key`, `value`, namespace, `group`, file_name, " +
		" create_time, create_by, modify_time, modify_by) values "
	values := []string{}
	args := []interface{}{}
	for k, v := range file.Metadata {
		values = append(values, " (?,?,?,?,?,sysdate(),?,sysdate(),?) ")
		args = append(args, k, v, file.Namespace, file.Group, file.Name, file.CreateBy, file.ModifyBy)
	}

	_, err := tx.Exec(insertSql, args...)
	return store.Error(err)
}

func (cf *configFileStore) batchCleanTags(tx *BaseTx, file *model.ConfigFile) error {
	// 添加配置标签
	cleanSql := "DELETE FROM config_file_tag WHERE namespace = ? AND `group` = ? AND file_name = ? "
	args := []interface{}{file.Namespace, file.Group, file.Name}
	_, err := tx.Exec(cleanSql, args...)
	return store.Error(err)
}

// CountConfigFiles 获取一个配置文件组下的文件数量
func (cfr *configFileStore) CountConfigFiles(namespace, group string) (uint64, error) {
	metricsSql := "SELECT count(file_name) FROM config_file WHERE flag = 0 AND namespace = ? AND `group` = ?"
	row := cfr.slave.QueryRow(metricsSql, namespace, group)
	var total uint64
	if err := row.Scan(&total); err != nil {
		return 0, store.Error(err)
	}
	return total, nil
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) GetConfigFile(namespace, group, name string) (*model.ConfigFile, error) {
	tx, err := cf.master.Begin()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	return cf.GetConfigFileTx(NewSqlDBTx(tx), namespace, group, name)
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) GetConfigFileTx(tx store.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	dbTx := tx.GetDelegateTx().(*BaseTx)
	querySql := cf.baseSelectConfigFileSql() + "where namespace = ? and `group` = ? and name = ? and flag = 0"
	rows, err := dbTx.Query(querySql, namespace, group, name)
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

// UpdateConfigFile 更新配置文件
func (cf *configFileStore) UpdateConfigFileTx(tx store.Tx, file *model.ConfigFile) error {
	updateSql := "update config_file set content = ? , comment = ?, format = ?, modify_time = sysdate(), " +
		" modify_by = ? where namespace = ? and `group` = ? and name = ?"
	dbTx := tx.GetDelegateTx().(*BaseTx)
	_, err := dbTx.Exec(updateSql, file.Content, file.Comment, file.Format,
		file.ModifyBy, file.Namespace, file.Group, file.Name)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// DeleteConfigFileTx 删除配置文件
func (cf *configFileStore) DeleteConfigFileTx(tx store.Tx, namespace, group, name string) error {
	deleteSql := "update config_file set flag = 1 where namespace = ? and `group` = ? and name = ?"
	var err error
	if tx != nil {
		_, err = tx.GetDelegateTx().(*BaseTx).Exec(deleteSql, namespace, group, name)
	} else {
		_, err = cf.master.Exec(deleteSql, namespace, group, name)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
func (cf *configFileStore) QueryConfigFiles(filter map[string]string, offset, limit uint32) (uint32, []*model.ConfigFile, error) {

	countSql := "SELECT COUNT(*) FROM config_file WHERE flag = 0 "
	querySql := cf.baseSelectConfigFileSql() + " WHERE flag = 0 "

	args := make([]interface{}, 0, len(filter))
	searchQuery := make([]string, 0, len(filter))

	for k, v := range filter {
		v = utils.ParseWildNameForSql(v)
		if utils.IsWildName(v) {
			searchQuery = append(searchQuery, k+" LIKE ? ")
		} else {
			searchQuery = append(searchQuery, k+" = ? ")
		}
		args = append(args, v)
	}

	countSql = countSql + (strings.Join(searchQuery, " AND "))
	var count uint32
	err := cf.master.QueryRow(countSql, args...).Scan(&count)
	if err != nil {
		return 0, nil, err
	}

	querySql = querySql + (strings.Join(searchQuery, " AND ")) + " ORDER BY id DESC LIMIT ?, ? "
	args = append(args, offset, limit)
	rows, err := cf.master.Query(querySql, args...)
	if err != nil {
		return 0, nil, err
	}

	files, err := cf.transferRows(rows)
	if err != nil {
		return 0, nil, err
	}

	return count, files, nil
}

// CountConfigFileEachGroup
func (cf *configFileStore) CountConfigFileEachGroup() (map[string]map[string]int64, error) {
	metricsSql := "SELECT namespace, `group`, count(name) FROM config_file WHERE flag = 0 GROUP by namespace, `group`"
	rows, err := cf.slave.Query(metricsSql)
	if err != nil {
		return nil, store.Error(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	ret := map[string]map[string]int64{}
	for rows.Next() {
		var (
			namespce string
			group    string
			cnt      int64
		)

		if err := rows.Scan(&namespce, &group, &cnt); err != nil {
			return nil, err
		}
		if _, ok := ret[namespce]; !ok {
			ret[namespce] = map[string]int64{}
		}
		ret[namespce][group] = cnt
	}

	return ret, nil
}

func (cf *configFileStore) baseSelectConfigFileSql() string {
	return "SELECT id, name,namespace,`group`,content,IFNULL(comment, ''),format, UNIX_TIMESTAMP(create_time), " +
		" IFNULL(create_by, ''),UNIX_TIMESTAMP(modify_time),IFNULL(modify_by, '') FROM config_file "
}

func (cf *configFileStore) hardDeleteConfigFile(namespace, group, name string) error {
	log.Infof("[Config][Storage] delete config file. namespace = %s, group = %s, name = %s", namespace, group, name)

	deleteSql := "delete from config_file where namespace = ? and `group` = ? and name = ? and flag = 1"

	_, err := cf.master.Exec(deleteSql, namespace, group, name)
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
		err := rows.Scan(&file.Id, &file.Name, &file.Namespace, &file.Group, &file.Content, &file.Comment,
			&file.Format, &ctime, &file.CreateBy, &mtime, &file.ModifyBy)
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
