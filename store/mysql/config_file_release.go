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
	"encoding/json"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.ConfigFileReleaseStore = (*configFileReleaseStore)(nil)

type configFileReleaseStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateConfigFileRelease 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileReleaseTx(tx store.Tx,
	data *model.ConfigFileRelease) error {

	s := "INSERT INTO config_file_release(name, namespace, `group`, file_name, content, " +
		" comment, md5, version, create_time, create_by, modify_time, modify_by) VALUES" +
		"(?,?,?,?,?,?,?,?, sysdate(),?,sysdate(),?)"

	dbTx := tx.GetDelegateTx().(*BaseTx)
	_, err := dbTx.Exec(s, data.Name, data.Namespace, data.Group,
		data.FileName, data.Content, data.Comment, data.Md5, data.Version,
		data.CreateBy, data.ModifyBy)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// GetConfigFileRelease 获取配置文件发布，只返回 flag=0 的记录
func (cfr *configFileReleaseStore) GetConfigFileRelease(
	req *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {

	querySql := cfr.baseQuerySql() + "WHERE namespace = ? AND `group` = ? AND " +
		" file_name = ? AND name = ? AND flag = 0 "
	var (
		rows *sql.Rows
		err  error
	)

	rows, err = cfr.master.Query(querySql, req.Namespace, req.Group, req.FileName, req.Name)
	if err != nil {
		return nil, err
	}
	fileRelease, err := cfr.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(fileRelease) > 0 {
		return fileRelease[0], nil
	}
	return nil, nil
}

// DeleteConfigFileRelease
func (cfr *configFileReleaseStore) DeleteConfigFileRelease(data *model.ConfigFileReleaseKey) error {
	s := "update config_file_release set flag = 1, modify_time = sysdate() " +
		" where namespace = ? and `group` = ? and file_name = ? and name = ?"
	_, err := cfr.master.Exec(s, data.Namespace, data.Group, data.FileName, data.Name)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// CleanConfigFileReleasesTx
func (cfr *configFileReleaseStore) CleanConfigFileReleasesTx(tx store.Tx,
	namespace, group, fileName string) error {

	dbTx := tx.GetDelegateTx().(*BaseTx)
	s := "UPDATE config_file_release SET flag = 1, modify_time = sysdate() WHERE namespace = ? " +
		" AND `group` = ? AND file_name = ?"
	_, err := dbTx.Exec(s, namespace, group, fileName)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// CleanDeletedConfigFileRelease 清理配置发布历史
func (cfr *configFileReleaseStore) CleanDeletedConfigFileRelease(endTime time.Time, limit uint64) error {
	delSql := "DELETE FROM config_file_release WHERE modify_time < ? AND flag = 1 LIMIT ?"
	_, err := cfr.master.Exec(delSql, endTime, limit)
	return err
}

// ActiveConfigFileRelease
func (cfr *configFileReleaseStore) ActiveConfigFileRelease(release *model.ConfigFileRelease) error {
	return cfr.master.processWithTransaction("ActiveConfigFileRelease", func(tx *BaseTx) error {
		args := []interface{}{release.Namespace, release.Group, release.FileName}
		if _, err := tx.Exec("SELECT * FROM config_file WHERE namespace = ? AND `group` = ? AND "+
			" name = ? FOR UPDATE", args...); err != nil {
			return err
		}
		maxVersion, err := cfr.inactiveConfigFileRelease(tx, release)
		if err != nil {
			return err
		}
		args = []interface{}{maxVersion + 1, release.Namespace, release.Group,
			release.FileName, release.Name}
		//	update 指定的 release 记录，设置其 active、version 以及 mtime
		updateSql := "UPDATE config_file_release SET active = 1, version = ?, modify_time = sysdate() " +
			" WHERE namespace = ? AND `group` = ? AND file_name = ? AND name = ?"
		if _, err := tx.Exec(updateSql, args...); err != nil {
			return err
		}

		return tx.Commit()
	})
}

func (cfr *configFileReleaseStore) inactiveConfigFileRelease(tx *BaseTx,
	release *model.ConfigFileRelease) (uint64, error) {

	args := []interface{}{release.Namespace, release.Group, release.FileName}
	//	先取消所有 active == true 的记录
	if _, err := tx.Exec("UPDATE config_file_release SET active = 0, modify_time = sysdate() "+
		" WHERE namespace = ? AND `group` = ? AND file_name = ? AND active = 1", args...); err != nil {
		return 0, err
	}

	//	生成最新的 version 版本信息
	row := tx.QueryRow("SELECT IFNULL(MAX(`version`), 0) FROM config_file_release WHERE namespace = ? AND "+
		" `group` = ? AND file_name = ?", args...)
	var maxVersion uint64
	if err := row.Scan(&maxVersion); err != nil {
		return 0, err
	}
	return maxVersion, nil
}

// GetMoreReleaseFile 获取最近更新的配置文件发布, 此方法用于 cache 增量更新，需要注意 modifyTime 应为数据库时间戳
func (cfr *configFileReleaseStore) GetMoreReleaseFile(firstUpdate bool,
	modifyTime time.Time) ([]*model.ConfigFileRelease, error) {

	if firstUpdate {
		modifyTime = time.Time{}
	}

	s := cfr.baseQuerySql() + " WHERE modify_time > FROM_UNIXTIME(?)"
	rows, err := cfr.slave.Query(s, timeToTimestamp(modifyTime))
	if err != nil {
		return nil, err
	}
	releases, err := cfr.transferRows(rows)
	if err != nil {
		return nil, err
	}
	return releases, nil
}

// CountConfigReleases 获取一个配置文件组下的文件数量
func (cfr *configFileReleaseStore) CountConfigReleases(namespace, group string) (uint64, error) {
	metricsSql := "SELECT count(file_name) FROM config_file_release WHERE flag = 0 " +
		" AND namespace = ? AND `group` = ?"
	row := cfr.slave.QueryRow(metricsSql, namespace, group)
	var total uint64
	if err := row.Scan(&total); err != nil {
		return 0, store.Error(err)
	}
	return total, nil
}

func (cfr *configFileReleaseStore) baseQuerySql() string {
	return "SELECT id, name, namespace, `group`, file_name, content, IFNULL(comment, ''), " +
		" md5, version, UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''), UNIX_TIMESTAMP(modify_time), " +
		" IFNULL(modify_by, ''), flag, IFNULL(tags, ''), active FROM config_file_release "
}

func (cfr *configFileReleaseStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileRelease, error) {
	if rows == nil {
		return nil, nil
	}
	defer func() {
		_ = rows.Close()
	}()

	var fileReleases []*model.ConfigFileRelease

	for rows.Next() {
		fileRelease := &model.ConfigFileRelease{}
		var (
			ctime, mtime, active int64
			tags                 string
		)
		err := rows.Scan(&fileRelease.Id, &fileRelease.Name, &fileRelease.Namespace, &fileRelease.Group,
			&fileRelease.FileName, &fileRelease.Content,
			&fileRelease.Comment, &fileRelease.Md5, &fileRelease.Version, &ctime, &fileRelease.CreateBy,
			&mtime, &fileRelease.ModifyBy, &fileRelease.Flag, &tags, &active)
		if err != nil {
			return nil, err
		}
		fileRelease.CreateTime = time.Unix(ctime, 0)
		fileRelease.ModifyTime = time.Unix(mtime, 0)
		fileRelease.Active = active == 1
		fileRelease.Valid = fileRelease.Flag == 0
		fileRelease.Metadata = map[string]string{}
		_ = json.Unmarshal([]byte(tags), &fileRelease.Metadata)
		fileReleases = append(fileReleases, fileRelease)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return fileReleases, nil
}
