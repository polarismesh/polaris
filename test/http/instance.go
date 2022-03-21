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
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

/**
 * @brief 实例数组转JSON
 */
func JSONFromInstances(instances []*api.Instance) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, instance := range instances {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, instance)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))
	return buffer, nil
}

/**
 * @brief 创建实例
 */
func (c *Client) CreateInstances(instances []*api.Instance) (*api.BatchWriteResponse, error) {
	fmt.Printf("\ncreate instances\n")

	url := fmt.Sprintf("http://%v/naming/%v/instances", c.Address, c.Version)

	body, err := JSONFromInstances(instances)
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
		return nil, err
	}

	return checkCreateInstancesResponse(ret, instances)
}

/**
 * @brief 删除实例
 */
func (c *Client) DeleteInstances(instances []*api.Instance) error {
	fmt.Printf("\ndelete instances\n")

	url := fmt.Sprintf("http://%v/naming/%v/instances/delete", c.Address, c.Version)

	body, err := JSONFromInstances(instances)
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

/**
 * @brief 更新实例
 */
func (c *Client) UpdateInstances(instances []*api.Instance) error {
	fmt.Printf("\nupdate instances\n")

	url := fmt.Sprintf("http://%v/naming/%v/instances", c.Address, c.Version)

	body, err := JSONFromInstances(instances)
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

/**
 * @brief 查询实例
 */
func (c *Client) GetInstances(instances []*api.Instance) error {
	fmt.Printf("\nget instances\n")

	url := fmt.Sprintf("http://%v/naming/%v/instances", c.Address, c.Version)

	params := map[string][]interface{}{
		"service":   {instances[0].GetService().GetValue()},
		"namespace": {instances[0].GetNamespace().GetValue()},
	}

	url = c.CompleteURL(url, params)
	response, err := c.SendRequest("GET", url, nil)
	if err != nil {
		return err
	}

	ret, err := GetBatchQueryResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	if ret.GetCode() == nil || ret.GetCode().GetValue() != api.ExecuteSuccess {
		return errors.New("invalid batch code")
	}

	instancesSize := len(instances)

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(instancesSize) {
		return fmt.Errorf("invalid batch amount, expect %d, obtain %v", instancesSize, ret.GetAmount())
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(instancesSize) {
		return errors.New("invalid batch size")
	}

	collection := make(map[string]*api.Instance)
	for _, instance := range instances {
		collection[instance.GetId().GetValue()] = instance
	}

	items := ret.GetInstances()
	if items == nil || len(items) != instancesSize {
		return errors.New("invalid batch instances")
	}

	for _, item := range items {
		if correctItem, ok := collection[item.GetId().GetValue()]; ok {
			if result := compareInstance(correctItem, item); !result {
				return fmt.Errorf("invalid instance %v", item.GetId().GetValue())
			}
		} else {
			return fmt.Errorf("instance %v not found", item.GetId().GetValue())
		}
	}
	return nil
}

/**
 * @brief 检查创建实例的回复
 */
func checkCreateInstancesResponse(ret *api.BatchWriteResponse, instances []*api.Instance) (
	*api.BatchWriteResponse, error) {

	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New("invalid batch code")
	case ret.GetSize().GetValue() != uint32(len(instances)):
		return nil, errors.New("invalid batch size")
	case len(ret.GetResponses()) != len(instances):
		return nil, errors.New("invalid batch response")
	}

	return ret, checkInstancesResponseEntry(ret, instances)
}

/**
 * @brief 检查创建实例每个实例的信息
 */
func checkInstancesResponseEntry(ret *api.BatchWriteResponse, instances []*api.Instance) error {
	items := ret.GetResponses()
	for index, item := range items {
		instance := item.GetInstance()
		switch {
		case item.GetCode().GetValue() != api.ExecuteSuccess:
			return errors.New("invalid code")
		case item.GetInstance() == nil:
			return errors.New("empty instance")
		case item.GetInstance().GetId().GetValue() == "":
			return errors.New("invalid instance id")
		case instance.GetService().GetValue() != instances[index].GetService().GetValue():
			return errors.New("invalid service")
		case instance.GetNamespace().GetValue() != instances[index].GetNamespace().GetValue():
			return errors.New("invalid namespace")
		case instance.GetHost().GetValue() != instances[index].GetHost().GetValue():
			return errors.New("invalid host")
		case instance.GetPort().GetValue() != instances[index].GetPort().GetValue():
			return errors.New("invalid port")
		}
	}

	return nil
}

/**
 * @brief 比较instance是否相等
 */
func compareInstance(correctItem *api.Instance, item *api.Instance) bool {
	// #lizard forgives
	correctID := correctItem.GetId().GetValue()
	correctService := correctItem.GetService().GetValue()
	correctNamespace := correctItem.GetNamespace().GetValue()
	correctHost := correctItem.GetHost().GetValue()
	correctPort := correctItem.GetPort().GetValue()
	correctProtocol := correctItem.GetProtocol().GetValue()
	correctVersion := correctItem.GetVersion().GetValue()
	correctPriority := correctItem.GetPriority().GetValue()
	correctWeight := correctItem.GetWeight().GetValue()
	correctHealthType := correctItem.GetHealthCheck().GetType()
	correctHealthTTL := correctItem.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
	correctHealthy := correctItem.GetHealthy().GetValue()
	correctIsolate := correctItem.GetIsolate().GetValue()
	correctMeta := correctItem.GetMetadata()
	correctLogicSet := correctItem.GetLogicSet().GetValue()
	/*correctCmdbRegion := correctItem.GetLocation().GetRegion().GetValue()
	correctCmdbZone := correctItem.GetLocation().GetZone().GetValue()
	correctCmdbCampus := correctItem.GetLocation().GetCampus().GetValue()*/

	id := item.GetId().GetValue()
	service := item.GetService().GetValue()
	namespace := item.GetNamespace().GetValue()
	host := item.GetHost().GetValue()
	port := item.GetPort().GetValue()
	protocol := item.GetProtocol().GetValue()
	version := item.GetVersion().GetValue()
	priority := item.GetPriority().GetValue()
	weight := item.GetWeight().GetValue()
	healthType := item.GetHealthCheck().GetType()
	healthTTL := item.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
	healthy := item.GetHealthy().GetValue()
	isolate := item.GetIsolate().GetValue()
	meta := item.GetMetadata()
	logicSet := item.GetLogicSet().GetValue()
	/*cmdbRegion := item.GetLocation().GetRegion().GetValue()
	cmdbZone := item.GetLocation().GetZone().GetValue()
	cmdbCampus := item.GetLocation().GetCampus().GetValue()*/

	if correctID == id && correctService == service && correctNamespace == namespace && correctHost == host &&
		correctPort == port && correctProtocol == protocol && correctVersion == version &&
		correctPriority == priority && correctWeight == weight && correctHealthType == healthType &&
		correctHealthTTL == healthTTL && correctHealthy == healthy && correctIsolate == isolate &&
		reflect.DeepEqual(correctMeta, meta) && correctLogicSet == logicSet {
		return true
	}
	return false
}
