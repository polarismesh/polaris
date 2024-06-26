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
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commonmodel "github.com/polarismesh/polaris/common/model"
)

type ConfigFileBase struct {
	Namespace string `param:"tenant"`
	Group     string `param:"group"`
	DataId    string `param:"dataId"`
}

type ConfigFile struct {
	ConfigFileBase
	Content          string `param:"content"`
	Tag              string `param:"tag"`
	AppName          string `param:"appName"`
	BetaIps          string `param:"betaIps"`
	CasMd5           string `param:"casMd5"`
	Type             string `param:"type"`
	SrcUser          string `param:"srcUser"`
	Labels           string `param:config_tags`
	Description      string `param:desc`
	EncryptedDataKey string `param:"encryptedDataKey"`
}

func (i *ConfigFile) ToDeleteSpec() *config_manage.ConfigFile {
	return &config_manage.ConfigFile{
		Namespace: wrapperspb.String(ToPolarisNamespace(i.Namespace)),
		Group:     wrapperspb.String(i.Group),
		Name:      wrapperspb.String(i.DataId),
	}
}

func (i *ConfigFile) ToQuerySpec() *config_manage.ClientConfigFileInfo {
	return &config_manage.ClientConfigFileInfo{
		Namespace: wrapperspb.String(ToPolarisNamespace(i.Namespace)),
		Group:     wrapperspb.String(i.Group),
		FileName:  wrapperspb.String(i.DataId),
	}
}

func (i *ConfigFile) ToSpecConfigFile() *config_manage.ConfigFilePublishInfo {
	specFile := &config_manage.ConfigFilePublishInfo{
		Tags:               make([]*config_manage.ConfigFileTag, 0, 4),
		Namespace:          wrapperspb.String(ToPolarisNamespace(i.Namespace)),
		Group:              wrapperspb.String(i.Group),
		FileName:           wrapperspb.String(i.DataId),
		Content:            wrapperspb.String(i.Content),
		Format:             wrapperspb.String(i.Type),
		Comment:            wrapperspb.String(i.Description),
		ReleaseDescription: wrapperspb.String(i.Description),
	}

	isCipher := strings.HasPrefix(i.DataId, "cipher-") && i.DataId != "cipher-"
	if isCipher {
		specFile.Encrypted = wrapperspb.Bool(true)
		specFile.EncryptAlgo = wrapperspb.String(strings.Split(i.DataId, "-")[1])
		if i.EncryptedDataKey != "" {
			specFile.Tags = append(specFile.Tags, &config_manage.ConfigFileTag{
				Key:   wrapperspb.String(commonmodel.MetaKeyConfigFileDataKey),
				Value: wrapperspb.String(i.EncryptedDataKey),
			})
		}
	}

	return specFile
}

const (
	LineSeparatorRune = '\x01'
	WordSeparatorRune = '\x02'
)

var (
	InvalidProbeModify = &NacosError{
		ErrCode: int32(http.StatusBadRequest),
		ErrMsg:  "invalid probeModify",
	}
)

type ConfigWatchContext struct {
	Request       *restful.Request
	ClientVersion string
	Items         []*ConfigListenItem
}

func (cw *ConfigWatchContext) ToSpecWatch() *config_manage.ClientWatchConfigFileRequest {
	specWatch := &config_manage.ClientWatchConfigFileRequest{
		ClientIp:   wrapperspb.String(cw.Request.Request.RemoteAddr),
		WatchFiles: make([]*config_manage.ClientConfigFileInfo, 0, len(cw.Items)),
	}
	for i := range cw.Items {
		item := cw.Items[i]
		specWatch.WatchFiles = append(specWatch.WatchFiles, &config_manage.ClientConfigFileInfo{
			Namespace: wrapperspb.String(ToPolarisNamespace(item.Tenant)),
			Group:     wrapperspb.String(item.Group),
			FileName:  wrapperspb.String(item.DataId),
			Md5:       wrapperspb.String(item.Md5),
		})
	}

	return specWatch
}

func (cw *ConfigWatchContext) IsSupportLongPolling() (time.Duration, bool) {
	val := cw.Request.HeaderParameter("Long-Pulling-Timeout")
	if val == "" {
		return 0, false
	}
	timeoutMs, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}

	finalTimeout := int64(math.Max(10000, float64(timeoutMs-2000)))
	return time.Millisecond * time.Duration(finalTimeout), true
}

func (cw *ConfigWatchContext) IsNoHangUp() bool {
	val := cw.Request.HeaderParameter("Long-Pulling-Timeout-No-Hangup")
	return strings.Compare(val, "true") == 0
}

type ConfigListenItem struct {
	Tenant string `json:"tenant"`
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Md5    string `json:"md5"`
}

func (c *ConfigListenItem) Key() string {
	if c.Tenant == "" {
		return url.QueryEscape(c.DataId + "+" + c.Group)
	}
	return url.QueryEscape(c.DataId + "+" + c.Group + "+" + c.Tenant)
}

// ParseConfigListenContext parse the transport protocol, which has two formats (W for field delimiter, L for each data delimiter)
// old: D w G w MD5 l
// new: D w G w MD5 w T l.
func ParseConfigListenContext(req *restful.Request, configKeysString string) (*ConfigWatchContext, error) {
	watchCtx := &ConfigWatchContext{
		Request: req,
		Items:   make([]*ConfigListenItem, 0, 4),
	}
	if configKeysString == "" {
		return watchCtx, nil
	}

	start := 0
	tmpList := make([]string, 0, 3)
	for i := start; i < len(configKeysString); i++ {
		c := configKeysString[i]
		if c == WordSeparatorRune {
			tmpList = append(tmpList, configKeysString[start:i])
			start = i + 1
			if len(tmpList) > 3 {
				// Malformed message and return parameter error.
				return nil, InvalidProbeModify
			}
		} else if c == LineSeparatorRune {
			endValue := ""
			if start+1 <= i {
				endValue = configKeysString[start:i]
			}
			start = i + 1

			// If it is the old message, the last digit is MD5. The post-multi-tenant message is tenant
			if len(tmpList) == 2 {
				watchCtx.Items = append(watchCtx.Items, &ConfigListenItem{
					Group:  tmpList[1],
					DataId: tmpList[0],
					Md5:    endValue,
				})
			} else {
				watchCtx.Items = append(watchCtx.Items, &ConfigListenItem{
					Tenant: endValue,
					Group:  tmpList[1],
					DataId: tmpList[0],
					Md5:    tmpList[2],
				})
			}
			tmpList = tmpList[:0]
			// Protect malformed messages
			if len(watchCtx.Items) > 10000 {
				return nil, InvalidProbeModify
			}
		}
	}

	watchCtx.ClientVersion = req.HeaderParameter("Client-Version")
	if watchCtx.ClientVersion == "" {
		watchCtx.ClientVersion = "2.0.0"
	}

	return watchCtx, nil
}
