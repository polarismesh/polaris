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
 * CONDITIONS OF ANY KIND, either express or Serveried. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"context"
	"errors"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

// createConfigFileTags 创建配置文件标签，tags 格式：k1,v1,k2,v2,k3,v3...
func (s *Server) createConfigFileTags(ctx context.Context, namespace, group, fileName, operator string, tags ...string) error {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if len(tags)%2 != 0 {
		return errors.New("tags param must be key,value pair, like key1,value1,key2,value2")
	}

	// 1. 获取已存储的 tags
	storedTags, err := s.storage.QueryTagByConfigFile(namespace, group, fileName)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] query config file tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return err
	}

	if len(storedTags) == 0 {
		return s.doCreateConfigFileTags(ctx, namespace, group, fileName, operator, tags...)
	}

	// 2. 新增 tag，一个 key 可以有多个的 value
	storedTagMap := make(map[string][]string, len(storedTags))
	for _, tag := range storedTags {
		if storedTagMap[tag.Key] == nil {
			storedTagMap[tag.Key] = []string{tag.Value}
		} else {
			storedTagMap[tag.Key] = append(storedTagMap[tag.Key], tag.Value)
		}
	}

	newTagMap := make(map[string][]string, len(tags))
	var key string
	for idx, t := range tags {
		if idx%2 == 0 {
			key = t
		} else {
			if newTagMap[key] == nil {
				newTagMap[key] = []string{t}
			} else {
				newTagMap[key] = append(newTagMap[key], t)
			}
		}
	}

	var toCreateTags []string
	for key, newTagValues := range newTagMap {
		storedTagValues := storedTagMap[key]
		for _, newTagValue := range newTagValues {
			if storedTagValues == nil {
				toCreateTags = append(toCreateTags, key)
				toCreateTags = append(toCreateTags, newTagValue)
			}
			var existed = false
			for _, storedTagValue := range storedTagValues {
				if storedTagValue == newTagValue {
					existed = true
				}
			}
			if !existed {
				toCreateTags = append(toCreateTags, key)
				toCreateTags = append(toCreateTags, newTagValue)
			}
		}
	}
	err = s.doCreateConfigFileTags(ctx, namespace, group, fileName, operator, toCreateTags...)
	if err != nil {
		return err
	}

	// 3. 删除 tag
	var toDeleteTags []string
	for key, storedTagValues := range storedTagMap {
		newTagValues := newTagMap[key]
		for _, storedTagValue := range storedTagValues {
			if newTagValues == nil {
				toDeleteTags = append(toDeleteTags, key)
				toDeleteTags = append(toDeleteTags, storedTagValue)
			}
			var existed = false
			for _, newTagValue := range newTagValues {
				if storedTagValue == newTagValue {
					existed = true
				}
			}
			if !existed {
				toDeleteTags = append(toDeleteTags, key)
				toDeleteTags = append(toDeleteTags, storedTagValue)
			}
		}
	}
	err = s.doDeleteConfigFileTags(ctx, namespace, group, fileName, toDeleteTags...)
	if err != nil {
		return err
	}

	return nil
}

// QueryConfigFileByTags 通过标签查询配置文件,多个 tag 之间为或的关系, tags 格式：k1,v1,k2,v2,k3,v3...
func (s *Server) queryConfigFileByTags(ctx context.Context, namespace, group, fileName string, offset, limit uint32,
	tags ...string) (int, []*model.ConfigFileTag, error) {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if len(tags)%2 != 0 {
		return 0, nil, errors.New("tags param must be key,value pair, like key1,value1,key2,value2")
	}

	files, err := s.storage.QueryConfigFileByTag(namespace, group, fileName, tags...)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] query config file by tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return 0, nil, err
	}

	if len(files) == 0 {
		return 0, nil, nil
	}

	// 去重
	var distinctFiles []*model.ConfigFileTag
	for _, file := range files {
		if distinctFiles == nil {
			distinctFiles = append(distinctFiles, file)
		} else {
			existed := false
			for _, distinctFile := range distinctFiles {
				if distinctFile.Namespace == file.Namespace && distinctFile.Group == file.Group && distinctFile.FileName == file.FileName {
					existed = true
					break
				}
			}
			if !existed {
				distinctFiles = append(distinctFiles, file)
			}
		}
	}

	// 内存分页
	fileCount := len(distinctFiles)
	if int(offset) >= fileCount {
		return fileCount, nil, nil
	}

	var endIdx int
	if int(offset+limit) >= fileCount {
		endIdx = fileCount
	} else {
		endIdx = int(offset + limit)
	}

	return fileCount, files[offset:endIdx], nil
}

// QueryTagsByConfigFileWithAPIModels 查询标签，返回API对象
func (s *Server) queryTagsByConfigFileWithAPIModels(ctx context.Context, namespace, group, fileName string) ([]*api.ConfigFileTag, error) {
	tags, err := s.storage.QueryTagByConfigFile(namespace, group, fileName)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, nil
	}

	var tagAPIModels []*api.ConfigFileTag
	for _, tag := range tags {
		tagAPIModels = append(tagAPIModels, &api.ConfigFileTag{
			Key:   utils.NewStringValue(tag.Key),
			Value: utils.NewStringValue(tag.Value),
		})
	}
	return tagAPIModels, nil
}

// deleteTagByConfigFile 删除配置文件的所有标签
func (s *Server) deleteTagByConfigFile(ctx context.Context, namespace, group, fileName string) error {
	if err := s.storage.DeleteTagByConfigFile(s.getTx(ctx), namespace, group, fileName); err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.ConfigScope().Error("[Config][Service] query config file tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return err
	}
	return nil
}

func (s *Server) doCreateConfigFileTags(ctx context.Context, namespace, group, fileName, operator string, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	var key string
	for idx, t := range tags {
		if idx%2 == 0 {
			key = t
		} else {
			err := s.storage.CreateConfigFileTag(s.getTx(ctx), &model.ConfigFileTag{
				Key:       key,
				Value:     t,
				Namespace: namespace,
				Group:     group,
				FileName:  fileName,
				CreateBy:  operator,
				ModifyBy:  operator,
			})
			if err != nil {
				log.ConfigScope().Error("[Config][Service] create config file tag error.",
					zap.String("request-id", requestID),
					zap.String("namespace", namespace),
					zap.String("group", group),
					zap.String("fileName", fileName),
					zap.Error(err))
				return err
			}
		}
	}
	return nil
}

func (s *Server) doDeleteConfigFileTags(ctx context.Context, namespace, group, fileName string, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	var key string
	for idx, t := range tags {
		if idx%2 == 0 {
			key = t
		} else {
			err := s.storage.DeleteConfigFileTag(s.getTx(ctx), namespace, group, fileName, key, t)
			if err != nil {
				log.ConfigScope().Error("[Config][Service] delete config file tag error.",
					zap.String("request-id", requestID),
					zap.String("namespace", namespace),
					zap.String("group", group),
					zap.String("fileName", fileName),
					zap.Error(err))
				return err
			}
		}
	}
	return nil
}
