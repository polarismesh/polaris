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

type configFileReleaseHistoryStore struct {
	db *BaseDB
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(tx store.Tx,
	fileReleaseHistory *model.ConfigFileReleaseHistory) error {

	sql := "insert into config_file_release_history(name, namespace, `group`, file_name, content, comment, " +
		" md5, type, status, format, tags, " +
		"create_time, create_by, modify_time, modify_by) values " +
		"(?,?,?,?,?,?,?,?,?,?,?,sysdate(),?,sysdate(),?)"
	var err error
	if tx != nil {
		_, err = tx.GetDelegateTx().(*BaseTx).Exec(sql, fileReleaseHistory.Name, fileReleaseHistory.Namespace,
			fileReleaseHistory.Group, fileReleaseHistory.FileName, fileReleaseHistory.Content,
			fileReleaseHistory.Comment, fileReleaseHistory.Md5,
			fileReleaseHistory.Type, fileReleaseHistory.Status, fileReleaseHistory.Format, fileReleaseHistory.Tags,
			fileReleaseHistory.CreateBy, fileReleaseHistory.ModifyBy)
	} else {
		_, err = rh.db.Exec(sql, fileReleaseHistory.Name, fileReleaseHistory.Namespace,
			fileReleaseHistory.Group, fileReleaseHistory.FileName, fileReleaseHistory.Content,
			fileReleaseHistory.Comment, fileReleaseHistory.Md5,
			fileReleaseHistory.Type, fileReleaseHistory.Status, fileReleaseHistory.Format, fileReleaseHistory.Tags,
			fileReleaseHistory.CreateBy, fileReleaseHistory.ModifyBy)
	}
	if err != nil {
		return store.Error(err)
	}
	return nil
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(namespace, group, fileName string,
	offset, limit uint32, endId uint64) (uint32, []*model.ConfigFileReleaseHistory, error) {
	countSql := "select count(*) from config_file_release_history where "
	querySql := rh.genSelectSql() + " where "

	var queryParams []interface{}
	if namespace != "" {
		countSql += " namespace = ? and "
		querySql += " namespace = ? and "
		queryParams = append(queryParams, namespace)
	}
	if endId > 0 {
		countSql += " id < ? and "
		querySql += " id < ? and "
		queryParams = append(queryParams, endId)
	}

	countSql += "`group` like ? and file_name like ?"
	querySql += "`group` like ? and file_name like ? order by id desc limit ?, ?"
	queryParams = append(queryParams, "%"+group+"%")
	queryParams = append(queryParams, "%"+fileName+"%")

	var count uint32
	err := rh.db.QueryRow(countSql, queryParams...).Scan(&count)
	if err != nil {
		return 0, nil, err
	}

	queryParams = append(queryParams, offset)
	queryParams = append(queryParams, limit)
	rows, err := rh.db.Query(querySql, queryParams...)
	if err != nil {
		return 0, nil, err
	}

	fileReleaseHistories, err := rh.transferRows(rows)
	if err != nil {
		return 0, nil, err
	}

	return count, fileReleaseHistories, nil
}

func (rh *configFileReleaseHistoryStore) GetLatestConfigFileReleaseHistory(namespace, group,
	fileName string) (*model.ConfigFileReleaseHistory, error) {

	sql := rh.genSelectSql() + "where namespace = ? and `group` = ? and file_name = ? order by id desc limit 1"

	rows, err := rh.db.Query(sql, namespace, group, fileName)
	if err != nil {
		return nil, err
	}

	fileReleaseHistories, err := rh.transferRows(rows)
	if err != nil {
		return nil, err
	}

	if len(fileReleaseHistories) == 0 {
		return nil, nil
	}

	return fileReleaseHistories[0], nil
}

func (rh *configFileReleaseHistoryStore) genSelectSql() string {
	return "select id, name, namespace, `group`, file_name, content, IFNULL(comment, ''), md5, format, tags, type, " +
		" status, UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''), UNIX_TIMESTAMP(modify_time), " +
		"IFNULL(modify_by, '') from config_file_release_history "
}

func (rh *configFileReleaseHistoryStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileReleaseHistory, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var fileReleaseHistories []*model.ConfigFileReleaseHistory

	for rows.Next() {
		fileReleaseHistory := &model.ConfigFileReleaseHistory{}
		var ctime, mtime int64
		err := rows.Scan(&fileReleaseHistory.Id, &fileReleaseHistory.Name, &fileReleaseHistory.Namespace,
			&fileReleaseHistory.Group,
			&fileReleaseHistory.FileName, &fileReleaseHistory.Content,
			&fileReleaseHistory.Comment, &fileReleaseHistory.Md5, &fileReleaseHistory.Format,
			&fileReleaseHistory.Tags,
			&fileReleaseHistory.Type, &fileReleaseHistory.Status,
			&ctime, &fileReleaseHistory.CreateBy, &mtime, &fileReleaseHistory.ModifyBy)
		if err != nil {
			return nil, err
		}
		fileReleaseHistory.CreateTime = time.Unix(ctime, 0)
		fileReleaseHistory.ModifyTime = time.Unix(mtime, 0)

		fileReleaseHistories = append(fileReleaseHistories, fileReleaseHistory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return fileReleaseHistories, nil
}
