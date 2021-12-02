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
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

/**
 * @brief 实现BusinessStore接口
 */
type businessStore struct {
	db *BaseDB
}

/**
 * @brief 增加业务集
 */
func (bs *businessStore) AddBusiness(b *model.Business) error {
	if b.ID == "" || b.Name == "" {
		log.Errorf("[Store][database] add business missing some params: %+v", b)
		return fmt.Errorf("add Business missing some params, id %s, name %s", b.ID, b.Name)
	}

	str := `insert into business(id, name, token, owner, ctime, mtime) 
		values(?, ?, ?, ?, sysdate(), sysdate())`
	_, err := bs.db.Exec(str, b.ID, b.Name, b.Token, b.Owner)

	return err
}

/**
 * @brief 删除业务集
 */
func (bs *businessStore) DeleteBusiness(bid string) error {
	if bid == "" {
		log.Errorf("[Store][database] delete business missing id")
		return fmt.Errorf("add Business missing some params, bid %s", bid)
	}

	// 删除操作把对应的数据flag修改
	str := "update business set flag = 1, mtime = sysdate() where id = ?"
	_, err := bs.db.Exec(str, bid)

	return err
}

/**
 * @brief 更新业务集
 */
func (bs *businessStore) UpdateBusiness(b *model.Business) error {
	if b.ID == "" || b.Name == "" {
		log.Errorf("[Store][database] update business missing some params")
		return fmt.Errorf("update Business missing some params, id %s, name %s", b.ID, b.Name)
	}

	str := "update business set name = ?, owner = ?, mtime = sysdate() where id = ?"
	_, err := bs.db.Exec(str, b.Name, b.Owner, b.ID)

	return err
}

/**
 * @brief 更新业务集token
 */
func (bs *businessStore) UpdateBusinessToken(bid string, token string) error {
	if bid == "" || token == "" {
		log.Errorf("[Store][business] update business token missing some params")
		return fmt.Errorf("update Business Token missing some params, bid %s, token %s", bid, token)
	}

	str := "update business set token = ?, mtime = sysdate() where id = ?"
	_, err := bs.db.Exec(str, token, bid)

	return err
}

/**
 * @brief 获取owner下所有的业务集
 */
func (bs *businessStore) ListBusiness(owner string) ([]*model.Business, error) {
	if owner == "" {
		log.Errorf("[Store][business] list business missing owner")
		return nil, fmt.Errorf("list Business Mising param owner")
	}

	str := genBusinessSelectSQL() + " where owner like '%?%'"
	rows, err := bs.db.Query(str, owner)
	if err != nil {
		log.Errorf("[Store][database] list all business err: %s", err.Error())
		return nil, err
	}

	return businessFetchRows(rows)

}

/**
 * @brief 根据业务集ID获取业务集详情
 */
func (bs *businessStore) GetBusinessByID(id string) (*model.Business, error) {
	if id == "" {
		log.Errorf("[Store][business] get business missing id")
		return nil, fmt.Errorf("get Business missing param id")
	}

	str := genBusinessSelectSQL() + " where id = ?"
	rows, err := bs.db.Query(str, id)
	if err != nil {
		log.Errorf("[Store][database] get business by id query err: %s", err.Error())
		return nil, err
	}

	out, err := businessFetchRows(rows)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out[0], nil
}

/**
 * @brief 根据mtime获取增量数据
 */
func (bs *businessStore) GetMoreBusiness(mtime time.Time) ([]*model.Business, error) {
	str := genBusinessSelectSQL() + " where UNIX_TIMESTAMP(mtime) >= ?"
	rows, err := bs.db.Query(str, mtime.Unix())
	if err != nil {
		log.Errorf("[Store][database] get more business err: %s", err.Error())
		return nil, err
	}

	return businessFetchRows(rows)
}

// 生成business查询语句
func genBusinessSelectSQL() string {
	str := `select id, name, token, owner, flag, 
			UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) 
			from business `
	return str
}

/**
 * @brief 取出rows的数据
 */
func businessFetchRows(rows *sql.Rows) ([]*model.Business, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var ctime, mtime int64
	var out []*model.Business
	var flag int
	for rows.Next() {
		var tmp model.Business
		err := rows.Scan(
			&tmp.ID, &tmp.Name, &tmp.Token,
			&tmp.Owner, &flag, &ctime, &mtime)
		if err != nil {
			return nil, err
		}

		tmp.CreateTime = time.Unix(ctime, 0)
		tmp.ModifyTime = time.Unix(mtime, 0)
		tmp.Valid = true
		if flag == 1 {
			tmp.Valid = false
		}

		out = append(out, &tmp)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
