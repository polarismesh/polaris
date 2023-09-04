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

package boltdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	tblNameNamespace string = "namespace"
	OwnerAttribute   string = "owner"
	NameAttribute    string = "name"
)

type namespaceStore struct {
	handler BoltHandler
}

const (
	defaultNamespace = "default"
	polarisNamespace = "Polaris"
)

var (
	namespaceToToken = map[string]string{
		defaultNamespace: "e2e473081d3d4306b52264e49f7ce227",
		polarisNamespace: "2d1bfe5d12e04d54b8ee69e62494c7fd",
	}
	namespaceToComment = map[string]string{
		defaultNamespace: "Default Environment",
		polarisNamespace: "Polaris-server",
	}
)

// InitData initialize the namespace data
func (n *namespaceStore) InitData() error {
	namespaces := []string{defaultNamespace, polarisNamespace}
	for _, namespace := range namespaces {
		ns, err := n.GetNamespace(namespace)
		if err != nil {
			return err
		}
		if ns == nil {
			err = n.AddNamespace(&model.Namespace{
				Name:       namespace,
				Comment:    namespaceToComment[namespace],
				Token:      namespaceToToken[namespace],
				Owner:      "polaris",
				Valid:      true,
				CreateTime: time.Now(),
				ModifyTime: time.Now(),
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AddNamespace add a namespace
func (n *namespaceStore) AddNamespace(namespace *model.Namespace) error {
	if namespace.Name == "" {
		return errors.New("store add namespace name is empty")
	}

	// 先删除无效数据，再添加新数据
	if err := n.cleanNamespace(namespace.Name); err != nil {
		return err
	}

	tn := time.Now()

	namespace.CreateTime = tn
	namespace.ModifyTime = tn
	namespace.Valid = true
	return n.handler.SaveValue(tblNameNamespace, namespace.Name, n.toStore(namespace))
}

func (n *namespaceStore) cleanNamespace(name string) error {
	if err := n.handler.DeleteValues(tblNameNamespace, []string{name}); err != nil {
		log.Errorf("[Store][boltdb] delete invalid namespace error, %+v", err)
		return err
	}

	return nil
}

// UpdateNamespace update a namespace
func (n *namespaceStore) UpdateNamespace(namespace *model.Namespace) error {
	if namespace.Name == "" {
		return errors.New("store update namespace name is empty")
	}
	properties := make(map[string]interface{})
	properties["Owner"] = namespace.Owner
	properties["Comment"] = namespace.Comment
	properties["ModifyTime"] = time.Now()
	properties["ServiceExportTo"] = utils.MustJson(namespace.ServiceExportTo)
	return n.handler.UpdateValue(tblNameNamespace, namespace.Name, properties)
}

// UpdateNamespaceToken update the token of a namespace
func (n *namespaceStore) UpdateNamespaceToken(name string, token string) error {
	if name == "" || token == "" {
		return fmt.Errorf(
			"store update namespace token some param are empty, name is %s, token is %s", name, token)
	}
	properties := make(map[string]interface{})
	properties["Token"] = token
	properties["ModifyTime"] = time.Now()
	return n.handler.UpdateValue(tblNameNamespace, name, properties)
}

// GetNamespace query namespace by name
func (n *namespaceStore) GetNamespace(name string) (*model.Namespace, error) {
	values, err := n.handler.LoadValues(tblNameNamespace, []string{name}, &Namespace{})
	if err != nil {
		return nil, err
	}
	nsValue, ok := values[name]
	if !ok {
		return nil, nil
	}
	ns := nsValue.(*Namespace)
	return n.toModel(ns), nil
}

type NamespaceSlice []*model.Namespace

// Len length of namespace slice
func (ns NamespaceSlice) Len() int {
	return len(ns)
}

// Less compare namespace
func (ns NamespaceSlice) Less(i, j int) bool {
	return ns[i].ModifyTime.Before(ns[j].ModifyTime)
}

// Swap swap elements
func (ns NamespaceSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func matchFieldValue(value string, pattern string) bool {
	if utils.IsWildName(pattern) {
		return utils.IsWildMatch(value, pattern)
	}
	return pattern == value
}

func matchFieldValueByPatterns(value string, patterns []string) bool {
	for _, p := range patterns {
		if matchFieldValue(value, p) {
			return true
		}
	}
	return false
}

// GetNamespaces get namespaces by offset and limit
func (n *namespaceStore) GetNamespaces(
	filter map[string][]string, offset, limit int) ([]*model.Namespace, uint32, error) {
	values, err := n.handler.LoadValuesAll(tblNameNamespace, &Namespace{})
	if err != nil {
		return nil, 0, err
	}
	namespaces := NamespaceSlice(n.toNamespaces(values))

	ret := make([]*model.Namespace, 0)
	for i := range namespaces {
		ns := namespaces[i]
		if !ns.Valid {
			continue
		}
		matched := true
		for index, patterns := range filter {
			if index == OwnerAttribute {
				if matched = matchFieldValueByPatterns(ns.Owner, patterns); !matched {
					break
				}
			}
			if index == NameAttribute {
				if matched = matchFieldValueByPatterns(ns.Name, patterns); !matched {
					break
				}
			}
		}
		if matched {
			ret = append(ret, ns)
		}
	}
	namespaces = ret

	sort.Sort(sort.Reverse(namespaces))
	startIdx := offset
	if startIdx >= len(namespaces) {
		return nil, uint32(len(namespaces)), nil
	}
	endIdx := startIdx + limit
	if endIdx > len(namespaces) {
		endIdx = len(namespaces)
	}
	ret = namespaces[startIdx:endIdx]
	return ret, uint32(len(namespaces)), nil
}

func (n *namespaceStore) toNamespaces(values map[string]interface{}) []*model.Namespace {
	namespaces := make([]*model.Namespace, 0, len(values))
	for _, nsValue := range values {
		namespaces = append(namespaces, n.toModel(nsValue.(*Namespace)))
	}
	return namespaces
}

// GetMoreNamespaces get the latest updated namespaces
func (n *namespaceStore) GetMoreNamespaces(mtime time.Time) ([]*model.Namespace, error) {
	values, err := n.handler.LoadValuesByFilter(
		tblNameNamespace, []string{"ModifyTime"}, &Namespace{}, func(value map[string]interface{}) bool {
			mTimeValue, ok := value["ModifyTime"]
			if !ok {
				return false
			}
			return mTimeValue.(time.Time).After(mtime)
		})
	if err != nil {
		return nil, err
	}
	return n.toNamespaces(values), nil
}

func (n *namespaceStore) toModel(data *Namespace) *model.Namespace {
	return toModelNamespace(data)
}

func toModelNamespace(data *Namespace) *model.Namespace {
	export := make(map[string]struct{})
	_ = json.Unmarshal([]byte(data.ServiceExportTo), &export)
	return &model.Namespace{
		Name:            data.Name,
		Comment:         data.Comment,
		Token:           data.Token,
		Owner:           data.Owner,
		ServiceExportTo: export,
		CreateTime:      data.CreateTime,
		ModifyTime:      data.ModifyTime,
		Valid:           data.Valid,
	}
}

func (n *namespaceStore) toStore(data *model.Namespace) *Namespace {
	return &Namespace{
		Name:            data.Name,
		Comment:         data.Comment,
		Token:           data.Token,
		Owner:           data.Owner,
		ServiceExportTo: utils.MustJson(data.ServiceExportTo),
		CreateTime:      data.CreateTime,
		ModifyTime:      data.ModifyTime,
		Valid:           data.Valid,
	}
}

// Namespace 命名空间结构体
type Namespace struct {
	Name    string
	Comment string
	Token   string
	Owner   string
	Valid   bool
	// ServiceExportTo 服务可见性设置
	ServiceExportTo string
	CreateTime      time.Time
	ModifyTime      time.Time
}
