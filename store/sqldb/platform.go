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
	"errors"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

// platformStore 的实现
type platformStore struct {
	master *BaseDB
}

// CreatePlatform 创建平台
func (p *platformStore) CreatePlatform(platform *model.Platform) error {
	if platform.ID == "" {
		return errors.New("create platform missing id")
	}

	if err := p.cleanPlatform(platform.ID); err != nil {
		return store.Error(err)
	}

	str := `insert into platform 
			(id, name, domain, qps, token, owner, department, comment, flag, ctime, mtime) 
			values(?,?,?,?,?,?,?,?,?,sysdate(),sysdate())`
	if _, err := p.master.Exec(str, platform.ID, platform.Name, platform.Domain, platform.QPS, platform.Token,
		platform.Owner, platform.Department, platform.Comment, 0); err != nil {
		log.Errorf("[Store][platform] create platform(%s) err: %s", platform.ID, err.Error())
		return store.Error(err)
	}

	return nil
}

// DeletePlatform 删除平台信息
func (p *platformStore) DeletePlatform(id string) error {
	if id == "" {
		return errors.New("delete platform missing id")
	}

	str := `update platform set flag = 1, mtime = sysdate() where id = ?`
	if _, err := p.master.Exec(str, id); err != nil {
		log.Errorf("[Store][platform] delete platform(%s) err: %s", id, err.Error())
		return store.Error(err)
	}

	return nil
}

// UpdatePlatform 修改平台信息
func (p *platformStore) UpdatePlatform(platform *model.Platform) error {
	str := `update platform set name = ?, domain = ?, qps = ?, token = ?, owner = ?, department = ?, comment = ?, 
			mtime = sysdate() where id = ?`
	if _, err := p.master.Exec(str, platform.Name, platform.Domain, platform.QPS, platform.Token, platform.Owner,
		platform.Department, platform.Comment, platform.ID); err != nil {
		log.Errorf("[Store][platform] update platform(%+v) err: %s", platform, err.Error())
		return store.Error(err)
	}
	return nil
}

// GetPlatformById 查询平台信息
func (p *platformStore) GetPlatformById(id string) (*model.Platform, error) {
	if id == "" {
		return nil, errors.New("get platform by id missing id")
	}

	str := genSelectPlatformSQL() + " where id = ? and flag = 0"

	rows, err := p.master.Query(str, id)
	if err != nil {
		log.Errorf("[Store][platform] get platform by id(%s) err: %s", id, err.Error())
		return nil, err
	}

	out, err := fetchPlatformRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out[0], nil
}

// GetPlatforms 根据过滤条件查询平台信息及总数
func (p *platformStore) GetPlatforms(filter map[string]string, offset uint32, limit uint32) (
	uint32, []*model.Platform, error) {
	out, err := p.getPlatforms(filter, offset, limit)
	if err != nil {
		return 0, nil, err
	}
	num, err := p.getPlatformsCount(filter)
	if err != nil {
		return 0, nil, err
	}
	return num, out, nil
}

// getPlatforms 根据过滤条件查询平台信息
func (p *platformStore) getPlatforms(filter map[string]string, offset uint32, limit uint32) (
	[]*model.Platform, error) {
	// 不查询任何内容，直接返回空数组
	if limit == 0 {
		return make([]*model.Platform, 0), nil
	}
	str := genSelectPlatformSQL() + " where flag = 0"
	filterStr, args := genFilterSQL(filter)
	if filterStr != "" {
		str += " and " + filterStr
	}

	order := &Order{"mtime", "desc"}
	page := &Page{offset, limit}
	opStr, opArgs := genOrderAndPage(order, page)

	str += opStr
	args = append(args, opArgs...)

	rows, err := p.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][platform] get platforms by filter query(%s) err: %s", str, err.Error())
		return nil, err
	}

	out, err := fetchPlatformRows(rows)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// getPlatformsCount 根据过滤条件获取平台总数
func (p *platformStore) getPlatformsCount(filter map[string]string) (uint32, error) {
	str := `select count(*) from platform where flag = 0 `
	filterStr, args := genFilterSQL(filter)
	if filterStr != "" {
		str += " and " + filterStr
	}
	return queryEntryCount(p.master, str, args)
}

// fetchPlatformRows 读取平台信息数据
func fetchPlatformRows(rows *sql.Rows) ([]*model.Platform, error) {
	defer rows.Close()
	var out []*model.Platform
	for rows.Next() {
		var platform model.Platform
		var flag int
		var ctime, mtime int64
		err := rows.Scan(&platform.ID, &platform.Name, &platform.Domain, &platform.QPS, &platform.Token,
			&platform.Owner, &platform.Department, &platform.Comment, &flag, &ctime, &mtime)
		if err != nil {
			log.Errorf("[Store][platform] fetch platform scan err: %s", err.Error())
			return nil, err
		}
		platform.CreateTime = time.Unix(ctime, 0)
		platform.ModifyTime = time.Unix(mtime, 0)
		platform.Valid = true
		if flag == 1 {
			platform.Valid = false
		}
		out = append(out, &platform)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][platform] fetch platform next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// cleanPlatform 彻底删除平台信息
func (p *platformStore) cleanPlatform(id string) error {
	str := `delete from platform where id = ? and flag = 1`
	if _, err := p.master.Exec(str, id); err != nil {
		log.Errorf("[Store][platform] clean platform (%s) err: %s", id, err.Error())
		return err
	}
	return nil
}

// genSelectPlatformSQL 查询平台信息sql
func genSelectPlatformSQL() string {
	str := `select id, name, domain, qps, token, owner, IFNULL(department, ""), IFNULL(comment, ""), flag, 
			unix_timestamp(ctime), unix_timestamp(mtime) from platform `
	return str
}
