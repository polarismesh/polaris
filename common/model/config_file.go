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

import (
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

/** ----------- DataObject ------------- */

// ConfigFileGroup 配置文件组数据持久化对象
type ConfigFileGroup struct {
	Id         uint64
	Name       string
	Namespace  string
	Comment    string
	Owner      string
	Business   string
	Department string
	Metadata   map[string]string
	CreateTime time.Time
	ModifyTime time.Time
	CreateBy   string
	ModifyBy   string
	Valid      bool
}

type ConfigFileKey struct {
	Name      string
	Namespace string
	Group     string
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
	Metadata   map[string]string
	Encrypt    bool
	Status     string
}

func (s *ConfigFile) Key() *ConfigFileKey {
	return &ConfigFileKey{
		Name:      s.Name,
		Namespace: s.Namespace,
		Group:     s.Group,
	}
}

func (s *ConfigFile) GetEncryptDataKey() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyDataKey]
	return val
}

func (s *ConfigFile) GetEncryptAlgo() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyEncryptAlgo]
	return val
}

func (s *ConfigFile) IsEncrypted() bool {
	return s.Encrypt || s.GetEncryptDataKey() != ""
}

// ConfigFileRelease 配置文件发布数据持久化对象
type ConfigFileRelease struct {
	*SimpleConfigFileRelease
	Content string
}

type ConfigFileReleaseKey struct {
	Id        uint64
	Name      string
	Namespace string
	Group     string
	FileName  string
}

func (c ConfigFileReleaseKey) ToFileKey() *ConfigFileKey {
	return &ConfigFileKey{
		Name:      c.FileName,
		Group:     c.Group,
		Namespace: c.Namespace,
	}
}

func (c ConfigFileReleaseKey) OwnerKey() string {
	return c.Namespace + "@" + c.Group
}

func (c ConfigFileReleaseKey) ActiveKey() string {
	return c.Namespace + "@" + c.Group + "@" + c.FileName
}

func (c ConfigFileReleaseKey) ReleaseKey() string {
	return c.Namespace + "@" + c.Group + "@" + c.FileName + "@" + c.Name
}

// SimpleConfigFileRelease 配置文件发布数据持久化对象
type SimpleConfigFileRelease struct {
	*ConfigFileReleaseKey
	Version    uint64
	Comment    string
	Md5        string
	Flag       int
	Active     bool
	Valid      bool
	Format     string
	Metadata   map[string]string
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
}

func (s SimpleConfigFileRelease) GetEncryptDataKey() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyDataKey]
	return val
}

func (s SimpleConfigFileRelease) GetEncryptAlgo() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyEncryptAlgo]
	return val
}

func (s SimpleConfigFileRelease) IsEncrypted() bool {
	return s.GetEncryptDataKey() != ""
}

// ConfigFileReleaseHistory 配置文件发布历史记录数据持久化对象
type ConfigFileReleaseHistory struct {
	Id         uint64
	Name       string
	Namespace  string
	Group      string
	FileName   string
	Format     string
	Metadata   map[string]string
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
	Reason     string
}

func (s ConfigFileReleaseHistory) GetEncryptDataKey() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyDataKey]
	return val
}

func (s ConfigFileReleaseHistory) GetEncryptAlgo() string {
	val, _ := s.Metadata[utils.ConfigFileTagKeyEncryptAlgo]
	return val
}

func (s ConfigFileReleaseHistory) IsEncrypted() bool {
	return s.GetEncryptDataKey() != ""
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

func ToConfigFileStore(file *config_manage.ConfigFile) *ConfigFile {
	var comment string
	if file.Comment != nil {
		comment = file.Comment.Value
	}
	var createBy string
	if file.CreateBy != nil {
		createBy = file.CreateBy.Value
	}
	var content string
	if file.Content != nil {
		content = file.Content.Value
	}
	var format string
	if file.Format != nil {
		format = file.Format.Value
	}
	return &ConfigFile{
		Name:      file.Name.GetValue(),
		Namespace: file.Namespace.GetValue(),
		Group:     file.Group.GetValue(),
		Content:   content,
		Comment:   comment,
		Format:    format,
		CreateBy:  createBy,
		Encrypt:   file.GetEncrypted().GetValue(),
	}
}

func ToConfigFileAPI(file *ConfigFile) *config_manage.ConfigFile {
	if file == nil {
		return nil
	}
	return &config_manage.ConfigFile{
		Id:         utils.NewUInt64Value(file.Id),
		Name:       utils.NewStringValue(file.Name),
		Namespace:  utils.NewStringValue(file.Namespace),
		Group:      utils.NewStringValue(file.Group),
		Content:    utils.NewStringValue(file.Content),
		Comment:    utils.NewStringValue(file.Comment),
		Format:     utils.NewStringValue(file.Format),
		CreateBy:   utils.NewStringValue(file.CreateBy),
		ModifyBy:   utils.NewStringValue(file.ModifyBy),
		CreateTime: utils.NewStringValue(commontime.Time2String(file.CreateTime)),
		ModifyTime: utils.NewStringValue(commontime.Time2String(file.ModifyTime)),
	}
}

// ToConfiogFileReleaseApi
func ToConfiogFileReleaseApi(release *ConfigFileRelease) *config_manage.ConfigFileRelease {
	if release == nil {
		return nil
	}

	return &config_manage.ConfigFileRelease{
		Id:         utils.NewUInt64Value(release.Id),
		Name:       utils.NewStringValue(release.Name),
		Namespace:  utils.NewStringValue(release.Namespace),
		Group:      utils.NewStringValue(release.Group),
		FileName:   utils.NewStringValue(release.FileName),
		Content:    utils.NewStringValue(release.Content),
		Comment:    utils.NewStringValue(release.Comment),
		Md5:        utils.NewStringValue(release.Md5),
		Version:    utils.NewUInt64Value(release.Version),
		CreateBy:   utils.NewStringValue(release.CreateBy),
		CreateTime: utils.NewStringValue(commontime.Time2String(release.CreateTime)),
		ModifyBy:   utils.NewStringValue(release.ModifyBy),
		ModifyTime: utils.NewStringValue(commontime.Time2String(release.ModifyTime)),
	}
}

// ToConfigFileReleaseStore
func ToConfigFileReleaseStore(release *config_manage.ConfigFileRelease) *ConfigFileRelease {
	if release == nil {
		return nil
	}
	var comment string
	if release.Comment != nil {
		comment = release.Comment.Value
	}
	var content string
	if release.Content != nil {
		content = release.Content.Value
	}
	var md5 string
	if release.Md5 != nil {
		md5 = release.Md5.Value
	}
	var version uint64
	if release.Version != nil {
		version = release.Version.Value
	}
	var createBy string
	if release.CreateBy != nil {
		createBy = release.CreateBy.Value
	}
	var modifyBy string
	if release.ModifyBy != nil {
		createBy = release.ModifyBy.Value
	}
	var id uint64
	if release.Id != nil {
		id = release.Id.Value
	}

	return &ConfigFileRelease{
		SimpleConfigFileRelease: &SimpleConfigFileRelease{
			ConfigFileReleaseKey: &ConfigFileReleaseKey{
				Id:        id,
				Namespace: release.Namespace.GetValue(),
				Group:     release.Group.GetValue(),
				FileName:  release.FileName.GetValue(),
			},
			Comment:  comment,
			Md5:      md5,
			Version:  version,
			CreateBy: createBy,
			ModifyBy: modifyBy,
		},
		Content: content,
	}
}

func ToReleaseHistoryAPI(releaseHistory *ConfigFileReleaseHistory) *config_manage.ConfigFileReleaseHistory {
	if releaseHistory == nil {
		return nil
	}
	return &config_manage.ConfigFileReleaseHistory{
		Id:         utils.NewUInt64Value(releaseHistory.Id),
		Name:       utils.NewStringValue(releaseHistory.Name),
		Namespace:  utils.NewStringValue(releaseHistory.Namespace),
		Group:      utils.NewStringValue(releaseHistory.Group),
		FileName:   utils.NewStringValue(releaseHistory.FileName),
		Content:    utils.NewStringValue(releaseHistory.Content),
		Comment:    utils.NewStringValue(releaseHistory.Comment),
		Format:     utils.NewStringValue(releaseHistory.Format),
		Tags:       FromTagMap(releaseHistory.Metadata),
		Md5:        utils.NewStringValue(releaseHistory.Md5),
		Type:       utils.NewStringValue(releaseHistory.Type),
		Status:     utils.NewStringValue(releaseHistory.Status),
		CreateBy:   utils.NewStringValue(releaseHistory.CreateBy),
		CreateTime: utils.NewStringValue(commontime.Time2String(releaseHistory.CreateTime)),
		ModifyBy:   utils.NewStringValue(releaseHistory.ModifyBy),
		ModifyTime: utils.NewStringValue(commontime.Time2String(releaseHistory.ModifyTime)),
	}
}

type kv struct {
	Key   string
	Value string
}

// FromTagJson 从 Tags Json 字符串里反序列化出 Tags
func FromTagMap(kvs map[string]string) []*config_manage.ConfigFileTag {
	tags := make([]*config_manage.ConfigFileTag, 0, len(kvs))
	for k, v := range kvs {
		tags = append(tags, &config_manage.ConfigFileTag{
			Key:   utils.NewStringValue(k),
			Value: utils.NewStringValue(v),
		})
	}

	return tags
}

func ToTagMap(tags []*ConfigFileTag) map[string]string {
	kvs := map[string]string{}
	for i := range tags {
		kvs[tags[i].Key] = tags[i].Value
	}

	return kvs
}

func ToConfigGroupAPI(group *ConfigFileGroup) *config_manage.ConfigFileGroup {
	if group == nil {
		return nil
	}
	return &config_manage.ConfigFileGroup{
		Id:         utils.NewUInt64Value(group.Id),
		Name:       utils.NewStringValue(group.Name),
		Namespace:  utils.NewStringValue(group.Namespace),
		Comment:    utils.NewStringValue(group.Comment),
		Owner:      utils.NewStringValue(group.Owner),
		CreateBy:   utils.NewStringValue(group.CreateBy),
		ModifyBy:   utils.NewStringValue(group.ModifyBy),
		CreateTime: utils.NewStringValue(commontime.Time2String(group.CreateTime)),
		ModifyTime: utils.NewStringValue(commontime.Time2String(group.ModifyTime)),
	}
}

func ToConfigGroupStore(group *config_manage.ConfigFileGroup) *ConfigFileGroup {
	var comment string
	if group.Comment != nil {
		comment = group.Comment.Value
	}
	var createBy string
	if group.CreateBy != nil {
		createBy = group.CreateBy.Value
	}
	var groupOwner string
	if group.Owner != nil && group.Owner.GetValue() != "" {
		groupOwner = group.Owner.GetValue()
	} else {
		groupOwner = createBy
	}
	return &ConfigFileGroup{
		Name:      group.Name.GetValue(),
		Namespace: group.Namespace.GetValue(),
		Comment:   comment,
		CreateBy:  createBy,
		Valid:     true,
		Owner:     groupOwner,
	}
}

func ToConfigFileTemplateAPI(template *ConfigFileTemplate) *config_manage.ConfigFileTemplate {
	return &config_manage.ConfigFileTemplate{
		Id:         utils.NewUInt64Value(template.Id),
		Name:       utils.NewStringValue(template.Name),
		Content:    utils.NewStringValue(template.Content),
		Comment:    utils.NewStringValue(template.Comment),
		Format:     utils.NewStringValue(template.Format),
		CreateBy:   utils.NewStringValue(template.CreateBy),
		CreateTime: utils.NewStringValue(commontime.Time2String(template.CreateTime)),
		ModifyBy:   utils.NewStringValue(template.ModifyBy),
		ModifyTime: utils.NewStringValue(commontime.Time2String(template.ModifyTime)),
	}
}

func ToConfigFileTemplateStore(template *config_manage.ConfigFileTemplate) *ConfigFileTemplate {
	return &ConfigFileTemplate{
		Id:       template.Id.GetValue(),
		Name:     template.Name.GetValue(),
		Content:  template.Content.GetValue(),
		Comment:  template.Comment.GetValue(),
		Format:   template.Format.GetValue(),
		CreateBy: template.CreateBy.GetValue(),
		ModifyBy: template.ModifyBy.GetValue(),
	}
}
