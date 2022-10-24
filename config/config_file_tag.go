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

package config

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// createConfigFileTags 创建配置文件标签，tags 格式：k1,v1,k2,v2,k3,v3...
func (s *Server) createConfigFileTags(ctx context.Context, namespace, group,
	fileName, operator string, tags ...string) error {

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if len(tags)%2 != 0 {
		return errors.New("tags param must be key,value pair, like key1,value1,key2,value2")
	}

	// 1. 获取已存储的 tags
	storedTags, err := s.storage.QueryTagByConfigFile(namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] query config file tags error.",
			utils.ZapRequestID(requestID),
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
	storedTagMap := make(map[string]map[string]struct{}, len(storedTags))
	for _, tag := range storedTags {
		if storedTagMap[tag.Key] == nil {
			storedTagMap[tag.Key] = map[string]struct{}{}
		}
		val := storedTagMap[tag.Key]

		val[tag.Value] = struct{}{}
		storedTagMap[tag.Key] = val
	}

	newTagMap := make(map[string]map[string]struct{}, len(tags))

	for i := 0; i < len(tags)-1; i += 2 {
		key := tags[i]
		if newTagMap[key] == nil {
			newTagMap[key] = map[string]struct{}{}
		}

		val := newTagMap[key]
		val[tags[i+1]] = struct{}{}

		newTagMap[key] = val
	}

	toCreateTags := diffTags(newTagMap, storedTagMap)
	if err = s.doCreateConfigFileTags(ctx, namespace, group, fileName, operator, toCreateTags...); err != nil {
		return err
	}

	// 3. 删除 tag
	toDeleteTags := diffTags(storedTagMap, newTagMap)
	if err = s.doDeleteConfigFileTags(ctx, namespace, group, fileName, toDeleteTags...); err != nil {
		return err
	}

	return nil
}

// diffTags Compare data from A and B more than B.
func diffTags(a, b map[string]map[string]struct{}) []string {
	tmp := make(map[string]map[string]struct{})

	for key, values := range a {

		existVals := b[key]
		if len(existVals) == 0 {
			tmp[key] = a[key]
			continue
		}

		if _, ok := tmp[key]; !ok {
			tmp[key] = map[string]struct{}{}
		}

		for val := range values {
			_, existed := existVals[val]
			if !existed {
				tmp[key][val] = struct{}{}
			}
		}
	}

	ret := make([]string, 0, 4)

	for k, vs := range tmp {

		for v := range vs {
			ret = append(ret, k, v)
		}

	}

	return ret
}

// QueryConfigFileByTags Inquire the configuration file through the label, the relationship between multiple TAGs,
//
//	TAGS format: K1, V1, K2, V2, K3, V3 ...
func (s *Server) queryConfigFileByTags(ctx context.Context, namespace, group, fileName string, offset, limit uint32,
	tags ...string) (int, []*model.ConfigFileTag, error) {

	if len(tags)%2 != 0 {
		return 0, nil, errors.New("tags param must be key,value pair, like key1,value1,key2,value2")
	}

	files, err := s.storage.QueryConfigFileByTag(namespace, group, fileName, tags...)
	if err != nil {
		log.Error("[Config][Service] query config file by tags error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return 0, nil, err
	}

	if len(files) == 0 {
		return 0, nil, nil
	}

	temp := make(map[string]struct{})
	ret := make([]*model.ConfigFileTag, 0, 4)

	for i := range files {
		file := files[i]

		k := fmt.Sprintf("%s@%s@%s", file.Namespace, file.Group, file.FileName)

		if _, ok := temp[k]; !ok {
			ret = append(ret, file)
			temp[k] = struct{}{}
		}
	}

	// 内存分页
	fileCount := len(ret)
	if int(offset) >= fileCount {
		return fileCount, nil, nil
	}

	var endIdx int
	if int(offset+limit) >= fileCount {
		endIdx = fileCount
	} else {
		endIdx = int(offset + limit)
	}

	return fileCount, ret[offset:endIdx], nil
}

// QueryTagsByConfigFileWithAPIModels 查询标签，返回API对象
func (s *Server) queryTagsByConfigFileWithAPIModels(ctx context.Context, namespace,
	group, fileName string) ([]*api.ConfigFileTag, error) {

	tags, err := s.storage.QueryTagByConfigFile(namespace, group, fileName)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, nil
	}

	tagAPIModels := make([]*api.ConfigFileTag, 0, len(tags))

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
		log.Error("[Config][Service] query config file tags error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return err
	}
	return nil
}

func (s *Server) doCreateConfigFileTags(ctx context.Context, namespace, group, fileName,
	operator string, tags ...string) error {

	if len(tags) == 0 {
		return nil
	}

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
				log.Error("[Config][Service] create config file tag error.",
					utils.ZapRequestIDByCtx(ctx),
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

	var key string
	for idx, t := range tags {
		if idx%2 == 0 {
			key = t
		} else {
			err := s.storage.DeleteConfigFileTag(s.getTx(ctx), namespace, group, fileName, key, t)
			if err != nil {
				log.Error("[Config][Service] delete config file tag error.",
					utils.ZapRequestIDByCtx(ctx),
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
