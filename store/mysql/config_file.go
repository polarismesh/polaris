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

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	configFileStoreFieldMapping = map[string]map[string]string{
		"config_file": {
			"group":     "`group`",
			"file_name": "name",
			"namespace": "namespace",
			"content":   "content",
		},
		"config_file_release": {
			"group":        "`group`",
			"file_name":    "file_name",
			"release_name": "name",
		},
		"config_file_group": {},
	}
)

var _ store.ConfigFileStore = (*configFileStore)(nil)

type configFileStore struct {
	master *BaseDB
	slave  *BaseDB
}

// LockConfigFile 加锁配置文件
func (cf *configFileStore) LockConfigFile(tx store.Tx, file *model.ConfigFileKey) (*model.ConfigFile, error) {
	if tx == nil {
		return nil, ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	args := []interface{}{file.Namespace, file.Group, file.Name}
	lockSql := cf.baseSelectConfigFileSql() +
		" WHERE namespace = ? AND `group` = ? AND name = ? AND flag = 0 FOR UPDATE"

	rows, err := dbTx.Query(lockSql, args...)
	if err != nil {
		return nil, store.Error(err)
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

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFileTx(tx store.Tx, file *model.ConfigFile) error {
	if tx == nil {
		return ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	deleteSql := "DELETE FROM config_file WHERE namespace = ? AND `group` = ? AND name = ? AND flag = 1"
	if _, err := dbTx.Exec(deleteSql, file.Namespace, file.Group, file.Name); err != nil {
		return store.Error(err)
	}

	createSql := "INSERT INTO config_file( " +
		" name, namespace, `group`, content, comment, format, create_time, " +
		"create_by, modify_time, modify_by) " +
		" VALUES " +
		"(?, ?, ?, ?, ?, ?, sysdate(), ?, sysdate(), ?)"
	if _, err := dbTx.Exec(createSql, file.Name, file.Namespace, file.Group,
		file.Content, file.Comment, file.Format, file.CreateBy, file.ModifyBy); err != nil {
		return store.Error(err)
	}

	if err := cf.batchCleanTags(dbTx, file); err != nil {
		return store.Error(err)
	}
	if err := cf.batchAddTags(dbTx, file); err != nil {
		return store.Error(err)
	}
	return nil
}

func (cf *configFileStore) batchAddTags(tx *BaseTx, file *model.ConfigFile) error {
	if len(file.Metadata) == 0 {
		return nil
	}

	// 添加配置标签
	insertSql := "INSERT INTO config_file_tag(" +
		" `key`, `value`, namespace, `group`, file_name, create_time, create_by, modify_time, modify_by) " +
		" VALUES "
	valuesSql := []string{}
	args := []interface{}{}
	for k, v := range file.Metadata {
		valuesSql = append(valuesSql, " (?, ?, ?, ?, ?, sysdate(), ?, sysdate(), ?) ")
		args = append(args, k, v, file.Namespace, file.Group, file.Name, file.CreateBy, file.ModifyBy)
	}
	insertSql = insertSql + strings.Join(valuesSql, ",")
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

func (cf *configFileStore) loadFileTags(tx *BaseTx, file *model.ConfigFile) error {
	querySql := "SELECT `key`, `value` FROM config_file_tag WHERE namespace = ? AND " +
		" `group` = ? AND file_name = ? "

	rows, err := tx.Query(querySql, file.Namespace, file.Group, file.Name)
	if err != nil {
		return err
	}
	if rows == nil {
		return nil
	}
	defer rows.Close()

	file.Metadata = make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		file.Metadata[key] = value
	}
	return nil
}

// CountConfigFiles 获取一个配置文件组下的文件数量
func (cfr *configFileStore) CountConfigFiles(namespace, group string) (uint64, error) {
	metricsSql := "SELECT count(*) FROM config_file WHERE flag = 0 AND namespace = ? AND `group` = ?"
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
func (cf *configFileStore) GetConfigFileTx(tx store.Tx,
	namespace, group, name string) (*model.ConfigFile, error) {
	if tx == nil {
		return nil, ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	querySql := cf.baseSelectConfigFileSql() + "WHERE namespace = ? AND `group` = ? AND name = ? AND flag = 0"
	rows, err := dbTx.Query(querySql, namespace, group, name)
	if err != nil {
		return nil, store.Error(err)
	}
	files, err := cf.transferRows(rows)
	if err != nil {
		return nil, store.Error(err)
	}
	if len(files) == 0 {
		return nil, nil
	}
	if err := cf.loadFileTags(dbTx, files[0]); err != nil {
		return nil, store.Error(err)
	}
	return files[0], nil
}

// UpdateConfigFile 更新配置文件
func (cf *configFileStore) UpdateConfigFileTx(tx store.Tx, file *model.ConfigFile) error {
	if tx == nil {
		return ErrTxIsNil
	}

	updateSql := "UPDATE config_file SET content = ?, comment = ?, format = ?, modify_time = sysdate(), " +
		" modify_by = ? WHERE namespace = ? AND `group` = ? AND name = ?"
	dbTx := tx.GetDelegateTx().(*BaseTx)
	_, err := dbTx.Exec(updateSql, file.Content, file.Comment, file.Format,
		file.ModifyBy, file.Namespace, file.Group, file.Name)
	if err != nil {
		return store.Error(err)
	}

	if err := cf.batchCleanTags(dbTx, file); err != nil {
		return store.Error(err)
	}
	if err := cf.batchAddTags(dbTx, file); err != nil {
		return store.Error(err)
	}
	return nil
}

// DeleteConfigFileTx 删除配置文件
func (cf *configFileStore) DeleteConfigFileTx(tx store.Tx, namespace, group, name string) error {
	if tx == nil {
		return ErrTxIsNil
	}

	deleteSql := "UPDATE config_file SET flag = 1 WHERE namespace = ? AND `group` = ? AND name = ?"
	dbTx := tx.GetDelegateTx().(*BaseTx)
	if _, err := dbTx.Exec(deleteSql, namespace, group, name); err != nil {
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
		if v, ok := configFileStoreFieldMapping["config_file"][k]; ok {
			k = v
		}
		if utils.IsWildName(v) {
			searchQuery = append(searchQuery, k+" LIKE ? ")
		} else {
			searchQuery = append(searchQuery, k+" = ? ")
		}
		args = append(args, utils.ParseWildNameForSql(v))
	}

	if len(searchQuery) > 0 {
		countSql = countSql + " AND "
		querySql = querySql + " AND "
	}
	countSql = countSql + (strings.Join(searchQuery, " AND "))

	var count uint32
	err := cf.master.QueryRow(countSql, args...).Scan(&count)
	if err != nil {
		log.Error("[Config][Storage] query config files", zap.String("count-sql", countSql), zap.Error(err))
		return 0, nil, store.Error(err)
	}

	querySql = querySql + (strings.Join(searchQuery, " AND ")) + " ORDER BY id DESC LIMIT ?, ? "

	args = append(args, offset, limit)
	rows, err := cf.master.Query(querySql, args...)
	if err != nil {
		log.Error("[Config][Storage] query config files", zap.String("query-sql", countSql), zap.Error(err))
		return 0, nil, store.Error(err)
	}

	files, err := cf.transferRows(rows)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	err = cf.slave.processWithTransaction("batch-load-file-tags", func(tx *BaseTx) error {
		for i := range files {
			item := files[i]
			if err := cf.loadFileTags(tx, item); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, store.Error(err)
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
	return "SELECT id, name, namespace, `group`, content, IFNULL(comment, ''), format, " +
		" UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''), UNIX_TIMESTAMP(modify_time), " +
		" IFNULL(modify_by, '') FROM config_file "
}

func (cf *configFileStore) hardDeleteConfigFile(namespace, group, name string) error {
	deleteSql := "DELETE FROM config_file WHERE namespace = ? AND `group` = ? AND name = ? AND flag = 1"
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

	var (
		files = make([]*model.ConfigFile, 0, 32)
	)

	for rows.Next() {
		file := &model.ConfigFile{
			Metadata: map[string]string{},
		}
		var ctime, mtime int64
		if err := rows.Scan(&file.Id, &file.Name, &file.Namespace, &file.Group, &file.Content, &file.Comment,
			&file.Format, &ctime, &file.CreateBy, &mtime, &file.ModifyBy); err != nil {
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
