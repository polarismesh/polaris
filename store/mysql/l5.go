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
	"time"

	"github.com/polarismesh/polaris/common/model"
)

// l5Store 实现了L5Store
type l5Store struct {
	master *BaseDB // 大部分操作都用主数据库
	slave  *BaseDB // 缓存相关的读取，请求到slave
}

// GetL5Extend 获取L5扩展数据
func (l5 *l5Store) GetL5Extend(serviceID string) (map[string]interface{}, error) {
	return nil, nil
}

// SetL5Extend 保存L5扩展数据
func (l5 *l5Store) SetL5Extend(serviceID string, meta map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// GenNextL5Sid 获取下一个sid
func (l5 *l5Store) GenNextL5Sid(layoutID uint32) (string, error) {
	var sid string
	var err error

	err = RetryTransaction("genNextL5Sid", func() error {
		sid, err = l5.genNextL5Sid(layoutID)
		return nil
	})

	return sid, err
}

// genNextL5Sid
func (l5 *l5Store) genNextL5Sid(layoutID uint32) (string, error) {
	tx, err := l5.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] get next l5 sid tx begin err: %s", err.Error())
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	getStr := "select module_id, interface_id, range_num from cl5_module limit 0, 1 for update"
	var mid, iid, rnum uint32
	if err := tx.QueryRow(getStr).Scan(&mid, &iid, &rnum); err != nil {
		log.Errorf("[Store][database] get next l5 sid err: %s", err.Error())
		return "", err
	}

	rnum++
	if rnum >= 65536 {
		rnum = 0
		iid++
	}
	if iid >= 4096 {
		iid = 1
		mid++
	}

	updateStr := "update cl5_module set module_id = ?, interface_id = ?, range_num = ?"
	if _, err := tx.Exec(updateStr, mid, iid, rnum); err != nil {
		log.Errorf("[Store][database] get next l5 sid, update module err: %s", err.Error())
		return "", err
	}
	// 更新完数据库之后，可以直接提交tx
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] get next l5 sid tx commit err: %s", err.Error())
		return "", err
	}

	// 数据表已经更改，生成sid的元素说明是唯一的，可以组合sid了
	modID := mid<<6 + layoutID
	cmdID := iid<<16 + rnum
	return fmt.Sprintf("%d:%d", modID, cmdID), nil
}

// GetMoreL5Extend 获取更多的增量数据
func (l5 *l5Store) GetMoreL5Extend(mtime time.Time) (map[string]map[string]interface{}, error) {
	return nil, nil
}

// GetMoreL5Routes 获取更多的L5 Route信息
func (l5 *l5Store) GetMoreL5Routes(flow uint32) ([]*model.Route, error) {
	str := getL5RouteSelectSQL() + " where Fflow > ?"
	rows, err := l5.slave.Query(str, flow)
	if err != nil {
		log.Errorf("[Store][database] get more l5 route query err: %s", err.Error())
		return nil, err
	}

	return l5RouteFetchRows(rows)
}

// GetMoreL5Policies 获取更多的L5 Policy信息
func (l5 *l5Store) GetMoreL5Policies(flow uint32) ([]*model.Policy, error) {
	str := getL5PolicySelectSQL() + " where Fflow > ?"
	rows, err := l5.slave.Query(str, flow)
	if err != nil {
		log.Errorf("[Store][database] get more l5 policy query err: %s", err.Error())
		return nil, err
	}

	return l5PolicyFetchRows(rows)
}

// GetMoreL5Sections 获取更多的L5 Section信息
func (l5 *l5Store) GetMoreL5Sections(flow uint32) ([]*model.Section, error) {
	str := getL5SectionSelectSQL() + " where Fflow > ?"
	rows, err := l5.slave.Query(str, flow)
	if err != nil {
		log.Errorf("[Store][database] get more l5 section query err: %s", err.Error())
		return nil, err
	}

	return l5SectionFetchRows(rows)
}

// GetMoreL5IPConfigs 获取更多的L5 IPConfig信息
func (l5 *l5Store) GetMoreL5IPConfigs(flow uint32) ([]*model.IPConfig, error) {
	str := getL5IPConfigSelectSQL() + " where Fflow > ?"
	rows, err := l5.slave.Query(str, flow)
	if err != nil {
		log.Errorf("[Store][database] get more l5 ip config query err: %s", err.Error())
		return nil, err
	}

	return l5IPConfigFetchRows(rows)
}

// getL5RouteSelectSQL 生成L5 Route的select sql语句
func getL5RouteSelectSQL() string {
	str := `select Fip, FmodId, FcmdId, FsetId, IFNULL(Fflag, 0), Fflow from t_route`
	return str
}

// getL5PolicySelectSQL 生成L5 Policy的select sql语句
func getL5PolicySelectSQL() string {
	str := `select FmodId, Fdiv, Fmod, IFNULL(Fflag, 0), Fflow from t_policy`
	return str
}

// getL5SectionSelectSQL 生成L5 Section的select sql语句
func getL5SectionSelectSQL() string {
	str := `select FmodId, Ffrom, Fto, Fxid, IFNULL(Fflag, 0), Fflow from t_section`
	return str
}

// getL5IPConfigSelectSQL 生成L5 IPConfig的select sql语句
func getL5IPConfigSelectSQL() string {
	str := `select Fip, FareaId, FcityId, FidcId, IFNULL(Fflag, 0), Fflow from t_ip_config`
	return str
}

// l5RouteFetchRows 从route中取出rows的数据
func l5RouteFetchRows(rows *sql.Rows) ([]*model.Route, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*model.Route
	var flag int

	progress := 0
	for rows.Next() {
		progress++
		if progress%100000 == 0 {
			log.Infof("[Store][database] load cl5 route progress: %d", progress)
		}
		space := &model.Route{}
		err := rows.Scan(
			&space.IP,
			&space.ModID,
			&space.CmdID,
			&space.SetID,
			&flag,
			&space.Flow)
		if err != nil {
			log.Errorf("[Store][database] fetch l5 route rows scan err: %s", err.Error())
			return nil, err
		}

		space.Valid = true
		if flag == 1 {
			space.Valid = false
		}

		out = append(out, space)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch l5 route rows next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// l5PolicyFetchRows 从policy中取出rows的数据
func l5PolicyFetchRows(rows *sql.Rows) ([]*model.Policy, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*model.Policy
	var flag int

	for rows.Next() {
		space := &model.Policy{}
		err := rows.Scan(
			&space.ModID,
			&space.Div,
			&space.Mod,
			&flag,
			&space.Flow)
		if err != nil {
			log.Errorf("[Store][database] fetch l5 policy rows scan err: %s", err.Error())
			return nil, err
		}

		space.Valid = true
		if flag == 1 {
			space.Valid = false
		}

		out = append(out, space)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch l5 policy rows next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// l5SectionFetchRows 从section中取出rows的数据
func l5SectionFetchRows(rows *sql.Rows) ([]*model.Section, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*model.Section
	var flag int

	for rows.Next() {
		space := &model.Section{}
		err := rows.Scan(
			&space.ModID,
			&space.From,
			&space.To,
			&space.Xid,
			&flag,
			&space.Flow)
		if err != nil {
			log.Errorf("[Store][database] fetch section rows scan err: %s", err.Error())
			return nil, err
		}

		space.Valid = true
		if flag == 1 {
			space.Valid = false
		}

		out = append(out, space)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch section rows next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// l5IPConfigFetchRows 从ip config中取出rows的数据
func l5IPConfigFetchRows(rows *sql.Rows) ([]*model.IPConfig, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*model.IPConfig
	var flag int
	for rows.Next() {
		space := &model.IPConfig{}
		err := rows.Scan(
			&space.IP,
			&space.AreaID,
			&space.CityID,
			&space.IdcID,
			&flag,
			&space.Flow)
		if err != nil {
			log.Errorf("[Store][database] fetch ip config rows scan err: %s", err.Error())
			return nil, err
		}

		space.Valid = true
		if flag == 1 {
			space.Valid = false
		}

		out = append(out, space)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch ip config rows next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}
