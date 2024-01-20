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
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type grayStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateGrayResourceTx 创建灰度资源
func (g *grayStore) CreateGrayResourceTx(tx store.Tx, data *model.GrayResource) error {
	if tx == nil {
		return ErrTxIsNil
	}
	dbTx := tx.GetDelegateTx().(*BaseTx)
	s := "INSERT INTO gray_resource(name, match_rule, create_time, create_by , modify_time, modify_by) " +
		" VALUES (?, ?, sysdate(), ? , sysdate(), ?) ON DUPLICATE KEY UPDATE " +
		"match_rule = ?, create_time=sysdate(), create_by=? , modify_time=sysdate(), modify_by=?"

	args := []interface{}{
		data.Name, data.MatchRule,
		data.CreateBy, data.ModifyBy,
		data.MatchRule, data.CreateBy, data.ModifyBy,
	}
	if _, err := dbTx.Exec(s, args...); err != nil {
		return store.Error(err)
	}
	return nil
}

func (g *grayStore) CleanGrayResource(tx store.Tx, data *model.GrayResource) error {
	if tx == nil {
		return ErrTxIsNil
	}
	dbTx := tx.GetDelegateTx().(*BaseTx)
	s := "UPDATE gray_resource SET flag = 1, modify_time = sysdate() WHERE name = ?"

	args := []interface{}{data.Name}
	if _, err := dbTx.Exec(s, args...); err != nil {
		return store.Error(err)
	}
	return nil
}

// GetMoreGrayResouces  获取最近更新的灰度资源, 此方法用于 cache 增量更新，需要注意 modifyTime 应为数据库时间戳
func (g *grayStore) GetMoreGrayResouces(firstUpdate bool,
	modifyTime time.Time) ([]*model.GrayResource, error) {

	if firstUpdate {
		modifyTime = time.Time{}
	}

	s := "SELECT name,  match_rule,  UNIX_TIMESTAMP(create_time), IFNULL(create_by, ''),  " +
		" UNIX_TIMESTAMP(modify_time), IFNULL(modify_by, ''), flag FROM gray_resource WHERE modify_time > FROM_UNIXTIME(?)"
	if firstUpdate {
		s += " AND flag = 0 "
	}
	rows, err := g.slave.Query(s, timeToTimestamp(modifyTime))
	if err != nil {
		return nil, err
	}
	grayResources, err := g.fetchGrayResourceRows(rows)
	if err != nil {
		return nil, err
	}
	return grayResources, nil
}

func (g *grayStore) fetchGrayResourceRows(rows *sql.Rows) ([]*model.GrayResource, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	grayResources := make([]*model.GrayResource, 0, 32)
	for rows.Next() {
		var ctime, mtime, valid int64
		grayResource := &model.GrayResource{}
		if err := rows.Scan(&grayResource.Name, &grayResource.MatchRule, &ctime,
			&grayResource.CreateBy, &mtime, &grayResource.ModifyBy, &valid); err != nil {
			return nil, err
		}
		grayResource.Valid = valid == 0
		grayResource.CreateTime = time.Unix(ctime, 0)
		grayResource.ModifyTime = time.Unix(mtime, 0)
		grayResources = append(grayResources, grayResource)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return grayResources, nil
}

// DeleteGrayResource 删除灰度资源
func (g *grayStore) DeleteGrayResource(tx store.Tx, data *model.GrayResource) error {
	s := "DELETE FROM  gray_resource  WHERE name= ?"
	_, err := g.master.Exec(s, data.Name)
	if err != nil {
		return store.Error(err)
	}
	return nil
}
