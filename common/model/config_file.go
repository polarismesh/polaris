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

package model

import "time"

/** ----------- DataObject ------------- */

// ConfigFileGroup 配置文件组数据持久化对象
type ConfigFileGroup struct {
	Id         uint64
	Name       string
	Namespace  string
	Comment    string
	CreateTime time.Time
	Owner      string
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Valid      bool
}

// ConfigFile 配置文件数据持久化对象
type ConfigFile struct {
	Id         uint64
	Name       string
	Namespace  string
	Group      string
	Content    string
	Comment    string
	Format     string
	Flag       int
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Valid      bool
}

// ConfigFileRelease 配置文件发布数据持久化对象
type ConfigFileRelease struct {
	Id         uint64
	Name       string
	Namespace  string
	Group      string
	FileName   string
	Content    string
	Comment    string
	Md5        string
	Version    uint64
	Flag       int
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Valid      bool
}

// ConfigFileReleaseHistory 配置文件发布历史记录数据持久化对象
type ConfigFileReleaseHistory struct {
	Id         uint64
	Name       string
	Namespace  string
	Group      string
	FileName   string
	Format     string
	Tags       string
	Content    string
	Comment    string
	Md5        string
	Type       string
	Status     string
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Valid      bool
}

// ConfigFileTag 配置文件标签数据持久化对象
type ConfigFileTag struct {
	Id         uint64
	Key        string
	Value      string
	Namespace  string
	Group      string
	FileName   string
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Valid      bool
}

// ConfigFileTemplate config file template data object
type ConfigFileTemplate struct {
	Id         uint64
	Name       string
	Content    string
	Comment    string
	Format     string
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
}
