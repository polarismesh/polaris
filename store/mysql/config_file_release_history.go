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
	"encoding/json"
	"strconv"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type configFileReleaseHistoryStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(
	history *model.ConfigFileReleaseHistory) error {

	s := "INSERT INTO config_file_release_history(" +
		" name, namespace, `group`, file_name, content, comment, md5, type, status, format, tags, " +
		"create_time, create_by, modify_time, modify_by, version, reason, description) " +
		" VALUES " +
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, sysdate(), ?, sysdate(), ?, ?, ?, ?)"
	_, err := rh.master.Exec(s, history.Name, history.Namespace,
		history.Group, history.FileName, history.Content,
		history.Comment, history.Md5,
		history.Type, history.Status, history.Format, utils.MustJson(history.Metadata),
		history.CreateBy, history.ModifyBy, history.Version, history.Reason, history.ReleaseDescription)
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(filter map[string]string,
	offset, limit uint32) (uint32, []*model.ConfigFileReleaseHistory, error) {
	countSql := "SELECT COUNT(*) FROM config_file_release_history WHERE "
	querySql := rh.genSelectSql() + " WHERE "

	namespace := filter["namespace"]
	group := filter["group"]
	fileName := filter["file_name"]
	endId, _ := strconv.ParseUint(filter["endId"], 10, 64)

	var queryParams []interface{}
	if namespace != "" {
		countSql += " namespace = ? AND "
		querySql += " namespace = ? AND "
		queryParams = append(queryParams, namespace)
	}
	if endId > 0 {
		countSql += " id < ? AND "
		querySql += " id < ? AND "
		queryParams = append(queryParams, endId)
	}

	countSql += " `group` LIKE ? AND file_name LIKE ? "
	querySql += " `group` LIKE ? AND file_name LIKE ? ORDER BY id DESC LIMIT ?, ? "
	queryParams = append(queryParams, "%"+group+"%")
	queryParams = append(queryParams, "%"+fileName+"%")

	var count uint32
	err := rh.master.QueryRow(countSql, queryParams...).Scan(&count)
	if err != nil {
		return 0, nil, err
	}

	queryParams = append(queryParams, offset)
	queryParams = append(queryParams, limit)
	rows, err := rh.master.Query(querySql, queryParams...)
	if err != nil {
		return 0, nil, err
	}

	fileReleaseHistories, err := rh.transferRows(rows)
	if err != nil {
		return 0, nil, err
	}

	return count, fileReleaseHistories, nil
}

// CleanConfigFileReleaseHistory 清理配置发布历史
func (rh *configFileReleaseHistoryStore) CleanConfigFileReleaseHistory(endTime time.Time, limit uint64) error {
	delSql := "DELETE FROM config_file_release_history WHERE create_time < ? LIMIT ?"
	_, err := rh.master.Exec(delSql, endTime, limit)
	return err
}

func (rh *configFileReleaseHistoryStore) genSelectSql() string {
	return "SELECT id, name, namespace, `group`, file_name, content, IFNULL(comment, ''), " +
		" md5, format, tags, type, status, UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''), " +
		" UNIX_TIMESTAMP(modify_time), IFNULL(modify_by, ''), IFNULL(reason, ''), " +
		" IFNULL(description, ''), IFNULL(version, 0) FROM config_file_release_history "
}

func (rh *configFileReleaseHistoryStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileReleaseHistory, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	records := make([]*model.ConfigFileReleaseHistory, 0, 16)

	for rows.Next() {
		item := &model.ConfigFileReleaseHistory{}
		var (
			ctime, mtime int64
			tags         string
		)
		err := rows.Scan(&item.Id, &item.Name, &item.Namespace, &item.Group, &item.FileName, &item.Content,
			&item.Comment, &item.Md5, &item.Format, &tags, &item.Type, &item.Status, &ctime, &item.CreateBy,
			&mtime, &item.ModifyBy, &item.Reason, &item.ReleaseDescription, &item.Version)
		if err != nil {
			return nil, err
		}
		item.CreateTime = time.Unix(ctime, 0)
		item.ModifyTime = time.Unix(mtime, 0)
		item.Metadata = map[string]string{}
		_ = json.Unmarshal([]byte(tags), &item.Metadata)

		records = append(records, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}
