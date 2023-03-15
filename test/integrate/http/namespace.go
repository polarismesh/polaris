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

package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/golang/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// JSONFromNamespaces 将命名空间数组转换为JSON
func JSONFromNamespaces(namespaces []*apimodel.Namespace) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, namespace := range namespaces {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, namespace)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))

	return buffer, nil
}

// CreateNamespaces 创建命名空间
func (c *Client) CreateNamespaces(namespaces []*apimodel.Namespace) (*apiservice.BatchWriteResponse, error) {
	fmt.Printf("\ncreate namespaces\n")

	url := fmt.Sprintf("http://%v/naming/%v/namespaces", c.Address, c.Version)

	body, err := JSONFromNamespaces(namespaces)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetBatchWriteResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return ret, err
	}

	return checkCreateNamespacesResponse(ret, namespaces)
}

// DeleteNamespaces 删除命名空间
func (c *Client) DeleteNamespaces(namespaces []*apimodel.Namespace) error {
	fmt.Printf("\ndelete namespaces\n")

	url := fmt.Sprintf("http://%v/naming/%v/namespaces/delete", c.Address, c.Version)

	body, err := JSONFromNamespaces(namespaces)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	_, err = GetBatchWriteResponse(response)
	if err != nil {
		if err == io.EOF {
			return nil
		}

		fmt.Printf("%v\n", err)
		return err
	}

	return nil
}

// DeleteNamespaces 删除命名空间
func (c *Client) DeleteNamespacesGetResp(namespaces []*apimodel.Namespace) (*apiservice.BatchWriteResponse, error) {
	fmt.Printf("\ndelete namespaces\n")

	url := fmt.Sprintf("http://%v/naming/%v/namespaces/delete", c.Address, c.Version)

	body, err := JSONFromNamespaces(namespaces)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	resp, err := GetBatchWriteResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return resp, err
	}

	return resp, nil
}

// UpdateNamesapces 更新命名空间
func (c *Client) UpdateNamesapces(namespaces []*apimodel.Namespace) error {
	fmt.Printf("\nupdate namespaces\n")

	url := fmt.Sprintf("http://%v/naming/%v/namespaces", c.Address, c.Version)

	body, err := JSONFromNamespaces(namespaces)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	response, err := c.SendRequest("PUT", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	_, err = GetBatchWriteResponse(response)
	if err != nil {
		if err == io.EOF {
			return nil
		}

		fmt.Printf("%v\n", err)
		return err
	}

	return nil
}

// GetNamespaces 查询命名空间
func (c *Client) GetNamespaces(namespaces []*apimodel.Namespace) ([]*apimodel.Namespace, error) {
	fmt.Printf("\nget namespaces\n")

	url := fmt.Sprintf("http://%v/naming/%v/namespaces", c.Address, c.Version)

	params := map[string][]interface{}{
		"name": {namespaces[0].GetName().GetValue(), namespaces[1].GetName().GetValue()},
	}

	url = c.CompleteURL(url, params)
	response, err := c.SendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ret, err := GetBatchQueryResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	if ret.GetCode() == nil || ret.GetCode().GetValue() != api.ExecuteSuccess {
		return nil, errors.New("invalid batch code")
	}

	namespacesSize := len(namespaces)

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(namespacesSize) {
		return nil, errors.New("invalid batch amount")
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(namespacesSize) {
		return nil, errors.New("invalid batch size")
	}

	collection := make(map[string]*apimodel.Namespace)
	for _, namespace := range namespaces {
		collection[namespace.GetName().GetValue()] = namespace
	}

	items := ret.GetNamespaces()
	if items == nil || len(items) != namespacesSize {
		return nil, errors.New("invalid batch namespaces")
	}

	for _, item := range items {
		if correctItem, ok := collection[item.GetName().GetValue()]; ok {
			if result := compareNamespace(correctItem, item); !result {
				return nil, errors.New("invalid namespace")
			}
		} else {
			return nil, errors.New("invalid namespace")
		}
	}
	return items, nil
}

/**
 * @brief 检查创建命名空间的回复
 */
func checkCreateNamespacesResponse(ret *apiservice.BatchWriteResponse, namespaces []*apimodel.Namespace) (
	*apiservice.BatchWriteResponse, error) {
	if ret.GetCode() == nil || ret.GetCode().GetValue() != api.ExecuteSuccess {
		return nil, errors.New("invalid batch code")
	}

	namespacesSize := len(namespaces)
	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(namespacesSize) {
		return nil, errors.New("invalid batch size")
	}

	items := ret.GetResponses()
	if items == nil || len(items) != namespacesSize {
		return nil, errors.New("invalid batch response")
	}

	for index, item := range items {
		if item.GetCode() == nil || item.GetCode().GetValue() != api.ExecuteSuccess {
			return nil, errors.New("invalid code")
		}

		namespace := item.GetNamespace()
		if namespace == nil {
			return nil, errors.New("empty namespace")
		}

		name := namespaces[index].GetName().GetValue()
		if namespace.GetName() == nil || namespace.GetName().GetValue() != name {
			return nil, errors.New("invalid namespace name")
		}

		if namespace.GetToken() == nil || namespace.GetToken().GetValue() == "" {
			return nil, errors.New("invalid namespace token")
		}
	}
	return ret, nil
}

/**
 * @brief 比较namespace是否相等
 */
func compareNamespace(correctItem *apimodel.Namespace, item *apimodel.Namespace) bool {
	correctName := correctItem.GetName().GetValue()
	correctComment := correctItem.GetComment().GetValue()

	name := item.GetName().GetValue()
	comment := item.GetComment().GetValue()

	if correctName == name && correctComment == comment {
		return true
	}
	return false
}
