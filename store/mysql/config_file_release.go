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
	"errors"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var _ store.ConfigFileReleaseStore = (*configFileReleaseStore)(nil)

var (
	ErrTxIsNil = errors.New("tx is nil")
)

type configFileReleaseStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateConfigFileRelease 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileReleaseTx(tx store.Tx, data *model.ConfigFileRelease) error {
	if tx == nil {
		return ErrTxIsNil
	}
	dbTx := tx.GetDelegateTx().(*BaseTx)
	args := []interface{}{data.Namespace, data.Group, data.Name}
	_, err := dbTx.Exec("SELECT id FROM config_file WHERE namespace = ? AND `group` = ? AND name = ? FOR UPDATE", args...)
	if err != nil {
		return store.Error(err)
	}

	clean := "DELETE FROM config_file_release WHERE namespace = ? AND `group` = ? AND file_name = ? AND name = ? AND flag = 1"
	if _, err := dbTx.Exec(clean, data.Namespace, data.Group, data.FileName, data.Name); err != nil {
		return store.Error(err)
	}

	maxVersion, err := cfr.inactiveConfigFileRelease(dbTx, data)
	if err != nil {
		return store.Error(err)
	}

	s := "INSERT INTO config_file_release(name, namespace, `group`, file_name, content , comment, md5, " +
		" version, create_time, create_by , modify_time, modify_by, active, tags, description, release_type) " +
		" VALUES (?, ?, ?, ?, ? , ?, ?, ?, sysdate(), ? , sysdate(), ?, 1, ?, ?, ?)"

	args = []interface{}{
		data.Name, data.Namespace, data.Group,
		data.FileName, data.Content, data.Comment, data.Md5, maxVersion + 1,
		data.CreateBy, data.ModifyBy, utils.MustJson(data.Metadata), data.ReleaseDescription, data.ReleaseType,
	}
	if _, err = dbTx.Exec(s, args...); err != nil {
		return store.Error(err)
	}
	return nil
}

// GetConfigFileRelease 获取配置文件发布，只返回 flag=0 的记录
func (cfr *configFileReleaseStore) GetConfigFileRelease(req *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {
	tx, err := cfr.master.Begin()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	return cfr.GetConfigFileReleaseTx(NewSqlDBTx(tx), req)
}

// GetConfigFileReleaseTx 在已开启的事务中获取配置文件发布内容，只获取 flag=0 的记录
func (cfr *configFileReleaseStore) GetConfigFileReleaseTx(tx store.Tx,
	req *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {
	if tx == nil {
		return nil, ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	querySql := cfr.baseQuerySql() + "WHERE namespace = ? AND `group` = ? AND " +
		" file_name = ? AND name = ? AND flag = 0 "
	var (
		rows *sql.Rows
		err  error
	)

	rows, err = dbTx.Query(querySql, req.Namespace, req.Group, req.FileName, req.Name)
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
func (cfr *configFileReleaseStore) DeleteConfigFileReleaseTx(tx store.Tx, data *model.ConfigFileReleaseKey) error {
	if tx == nil {
		return ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	s := "update config_file_release set flag = 1, modify_time = sysdate() " +
		" where namespace = ? and `group` = ? and file_name = ? and name = ?"
	_, err := dbTx.Exec(s, data.Namespace, data.Group, data.FileName, data.Name)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// CleanConfigFileReleasesTx
func (cfr *configFileReleaseStore) CleanConfigFileReleasesTx(tx store.Tx,
	namespace, group, fileName string) error {
	if tx == nil {
		return ErrTxIsNil
	}

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

// GetConfigFileActiveRelease .
func (cfr *configFileReleaseStore) GetConfigFileActiveRelease(file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	tx, err := cfr.master.Begin()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	return cfr.GetConfigFileActiveReleaseTx(NewSqlDBTx(tx), file)
}

// GetConfigFileActiveReleaseTx .
func (cfr *configFileReleaseStore) GetConfigFileActiveReleaseTx(tx store.Tx,
	file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	if tx == nil {
		return nil, ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	querySql := cfr.baseQuerySql() + "WHERE namespace = ? AND `group` = ? AND " +
		" file_name = ? AND active = 1 AND release_type = ? AND flag = 0 "
	var (
		rows *sql.Rows
		err  error
	)

	rows, err = dbTx.Query(querySql, file.Namespace, file.Group, file.Name, model.ReleaseTypeFull)
	if err != nil {
		return nil, err
	}
	fileRelease, err := cfr.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(fileRelease) > 1 {
		return nil, errors.New("multi active file release found")
	}
	if len(fileRelease) > 0 {
		return fileRelease[0], nil
	}
	return nil, nil
}

// GetConfigFileBetaReleaseTx .
func (cfr *configFileReleaseStore) GetConfigFileBetaReleaseTx(tx store.Tx,
	file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	if tx == nil {
		return nil, ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	querySql := cfr.baseQuerySql() + "WHERE namespace = ? AND `group` = ? AND " +
		" file_name = ? AND active = 1 AND release_type = ? AND flag = 0 "
	var (
		rows *sql.Rows
		err  error
	)

	rows, err = dbTx.Query(querySql, file.Namespace, file.Group, file.Name, model.ReleaseTypeGray)
	if err != nil {
		return nil, err
	}
	fileRelease, err := cfr.transferRows(rows)
	if err != nil {
		return nil, err
	}
	if len(fileRelease) > 1 {
		return nil, errors.New("multi active file release found")
	}
	if len(fileRelease) > 0 {
		return fileRelease[0], nil
	}
	return nil, nil
}

// ActiveConfigFileReleaseTx
func (cfr *configFileReleaseStore) ActiveConfigFileReleaseTx(tx store.Tx, release *model.ConfigFileRelease) error {
	if tx == nil {
		return ErrTxIsNil
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)
	maxVersion, err := cfr.inactiveConfigFileRelease(dbTx, release)
	if err != nil {
		return err
	}
	args := []interface{}{maxVersion + 1, release.ReleaseType, release.Namespace, release.Group,
		release.FileName, release.Name}
	//	update 指定的 release 记录，设置其 active、version 以及 mtime
	updateSql := "UPDATE config_file_release SET active = 1, version = ?, modify_time = sysdate(), release_type = ? " +
		" WHERE namespace = ? AND `group` = ? AND file_name = ? AND name = ?"
	if _, err := dbTx.Exec(updateSql, args...); err != nil {
		return store.Error(err)
	}
	return nil
}

func (cfr *configFileReleaseStore) InactiveConfigFileReleaseTx(tx store.Tx, release *model.ConfigFileRelease) error {
	if tx == nil {
		return ErrTxIsNil
	}
	dbTx := tx.GetDelegateTx().(*BaseTx)

	args := []interface{}{release.Namespace, release.Group, release.FileName, release.Name, release.ReleaseType}
	//	取消对应发布版本的 active 状态
	if _, err := dbTx.Exec("UPDATE config_file_release SET active = 0, modify_time = sysdate() "+
		" WHERE namespace = ? AND `group` = ? AND file_name = ? AND name = ? AND release_type = ?", args...); err != nil {
		return store.Error(err)
	}
	return nil
}

func (cfr *configFileReleaseStore) inactiveConfigFileRelease(tx *BaseTx,
	release *model.ConfigFileRelease) (uint64, error) {
	if tx == nil {
		return 0, ErrTxIsNil
	}

	args := []interface{}{release.Namespace, release.Group, release.FileName, release.ReleaseType}
	//	先取消所有 active == true 的记录
	if _, err := tx.Exec("UPDATE config_file_release SET active = 0, modify_time = sysdate() "+
		" WHERE namespace = ? AND `group` = ? AND file_name = ? AND active = 1 AND release_type = ?", args...); err != nil {
		return 0, err
	}
	return cfr.selectMaxVersion(tx, release)
}

func (cfr *configFileReleaseStore) selectMaxVersion(tx *BaseTx, release *model.ConfigFileRelease) (uint64, error) {
	if tx == nil {
		return 0, ErrTxIsNil
	}

	args := []interface{}{release.Namespace, release.Group, release.FileName, release.ReleaseType}
	//	生成最新的 version 版本信息
	row := tx.QueryRow("SELECT IFNULL(MAX(`version`), 0) FROM config_file_release WHERE namespace = ? AND "+
		" `group` = ? AND file_name = ?", args[:3]...)
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

	s := cfr.baseQuerySql() + " WHERE modify_time > FROM_UNIXTIME(?) ORDER BY version DESC"
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
func (cfr *configFileReleaseStore) CountConfigReleases(namespace, group string, onlyActive bool) (uint64, error) {
	metricsSql := "SELECT count(file_name) FROM config_file_release WHERE flag = 0 " +
		" AND namespace = ? AND `group` = ?"
	if onlyActive {
		metricsSql = "SELECT count(file_name) FROM config_file_release WHERE flag = 0 " +
			" AND namespace = ? AND `group` = ? AND active = 1"
	}
	row := cfr.master.QueryRow(metricsSql, namespace, group)
	var total uint64
	if err := row.Scan(&total); err != nil {
		return 0, store.Error(err)
	}
	return total, nil
}

func (cfr *configFileReleaseStore) baseQuerySql() string {
	return "SELECT id, name, namespace, `group`, file_name, content, IFNULL(comment, ''), " +
		" md5, version, UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''), UNIX_TIMESTAMP(modify_time), " +
		" IFNULL(modify_by, ''), flag, IFNULL(tags, ''), active, IFNULL(description, ''), IFNULL(release_type, '') FROM config_file_release "
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
		fileRelease := model.NewConfigFileRelease()
		var (
			ctime, mtime, active int64
			tags                 string
		)
		err := rows.Scan(&fileRelease.Id, &fileRelease.Name, &fileRelease.Namespace, &fileRelease.Group,
			&fileRelease.FileName, &fileRelease.Content,
			&fileRelease.Comment, &fileRelease.Md5, &fileRelease.Version, &ctime, &fileRelease.CreateBy,
			&mtime, &fileRelease.ModifyBy, &fileRelease.Flag, &tags, &active, &fileRelease.ReleaseDescription,
			&fileRelease.ReleaseType)
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
