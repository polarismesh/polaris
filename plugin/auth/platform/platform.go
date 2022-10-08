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

package platform

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName plugin name
	PluginName = "platform"
	// DefaultTimeDiff default time diff
	DefaultTimeDiff = -1 * time.Second * 5
)

// init 初始化注册函数
func init() {
	plugin.RegisterPlugin(PluginName, &Auth{})
}

// Auth 鉴权插件
type Auth struct {
	dbType       string
	dbSourceName string
	interval     time.Duration
	whiteList    string
	firstUpdate  bool
	lastMtime    time.Time
	ids          *sync.Map
}

// Name 返回插件名字
func (a *Auth) Name() string {
	return PluginName
}

// Initialize 初始化鉴权插件
func (a *Auth) Initialize(conf *plugin.ConfigEntry) error {
	dbType, _ := conf.Option["dbType"].(string)
	dbAddr, _ := conf.Option["dbAddr"].(string)
	dbName, _ := conf.Option["dbName"].(string)
	interval, _ := conf.Option["interval"].(int)
	whiteList, _ := conf.Option["white-list"].(string)

	if dbType == "" || dbAddr == "" || dbName == "" {
		return fmt.Errorf("config Plugin %s missing database params", PluginName)
	}

	if interval == 0 {
		return fmt.Errorf("config Plugin %s has error interval: %d", PluginName, interval)
	}

	a.dbType = dbType
	a.dbSourceName = dbAddr + "/" + dbName
	a.interval = time.Duration(interval) * time.Second
	a.whiteList = whiteList
	a.ids = new(sync.Map)
	a.lastMtime = time.Unix(0, 0)
	a.firstUpdate = true

	if err := a.update(); err != nil {
		log.Errorf("[Plugin][%s] update err: %s", PluginName, err.Error())
		return err
	}
	a.firstUpdate = false

	go a.run()
	return nil
}

// Destroy 销毁插件
func (a *Auth) Destroy() error {
	return nil
}

// Allow 判断请求是否允许通过
func (a *Auth) Allow(platformID, platformToken string) bool {
	if platformID == "" || platformToken == "" {
		return false
	}
	platform := a.getPlatformByID(platformID)
	if platform == nil {
		log.Errorf("[Plugin][%s] platform (%s) does not exist", PluginName, platformID)
		return false
	}

	return platform.Token == platformToken
}

// IsWhiteList 判断请求ip是否属于白名单
func (a *Auth) IsWhiteList(ip string) bool {
	if ip == "" || a.whiteList == "" {
		return false
	}

	return ip == a.whiteList
}

func (a *Auth) run() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for range ticker.C {
		_ = a.update()
	}
}

// update 更新数据
func (a *Auth) update() error {
	start := time.Now()
	out, err := a.getPlatforms()
	if err != nil {
		return err
	}

	update, del := a.setPlatform(out)
	log.Info("[Plugin][platform] get more platforms", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", a.lastMtime), zap.Duration("used", time.Since(start)))
	return nil
}

// getPlatforms 从数据库中拉取增量数据
func (a *Auth) getPlatforms() ([]*model.Platform, error) {
	// 每次采用短连接的方式，重新连接mysql
	db, err := sql.Open(a.dbType, a.dbSourceName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	str := `select id, qps, token, flag, unix_timestamp(mtime) from platform where mtime > ?`
	if a.firstUpdate {
		str += " and flag != 1" // nolint
	}
	rows, err := db.Query(str, commontime.Time2String(a.lastMtime.Add(DefaultTimeDiff)))
	if err != nil {
		log.Errorf("[Store][platform] query platform with mtime err: %s", err.Error())
		return nil, err
	}

	out, err := fetchPlatformRows(rows)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// fetchPlatformRows 读取平台信息数据
func fetchPlatformRows(rows *sql.Rows) ([]*model.Platform, error) {
	defer rows.Close()
	var out []*model.Platform
	for rows.Next() {
		var platform model.Platform
		var flag int
		var mtime int64
		err := rows.Scan(&platform.ID, &platform.QPS, &platform.Token, &flag, &mtime)
		if err != nil {
			log.Errorf("[Plugin][%s] fetch platform scan err: %s", PluginName, err.Error())
			return nil, err
		}
		platform.ModifyTime = time.Unix(mtime, 0)
		platform.Valid = true
		if flag == 1 {
			platform.Valid = false
		}
		out = append(out, &platform)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Plugin][%s] fetch platform next err: %s", PluginName, err.Error())
		return nil, err
	}
	return out, nil
}

// setPlatform 更新数据到缓存
func (a *Auth) setPlatform(platforms []*model.Platform) (int, int) {
	if len(platforms) == 0 {
		return 0, 0
	}

	lastMtime := a.lastMtime.Unix()
	update := 0
	del := 0
	for _, entry := range platforms {
		if entry.ID == "" {
			continue
		}

		if entry.ModifyTime.Unix() > lastMtime {
			lastMtime = entry.ModifyTime.Unix()
		}

		if !entry.Valid {
			del++
			a.ids.Delete(entry.ID)
			continue
		}

		update++
		a.ids.Store(entry.ID, entry)
	}

	if a.lastMtime.Unix() < lastMtime {
		a.lastMtime = time.Unix(lastMtime, 0)
	}
	return update, del
}

// getPlatformByID 根据平台ID获取平台信息
func (a *Auth) getPlatformByID(id string) *model.Platform {
	if id == "" {
		return nil
	}
	value, ok := a.ids.Load(id)
	if !ok {
		return nil
	}

	return value.(*model.Platform)
}
