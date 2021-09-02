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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/golang/protobuf/jsonpb"
	"io"
)

/**
 * @brief 平台数据转JSON
 */
func JSONFromPlatforms(platforms []*api.Platform) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, service := range platforms {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, service)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))
	return buffer, nil
}

/**
 * @brief 创建平台
 */
func (c *Client) CreatePlatforms(platforms []*api.Platform) (*api.BatchWriteResponse, error) {
	fmt.Printf("\ncreate platforms\n")

	url := fmt.Sprintf("http://%v/naming/%v/platforms", c.Address, c.Version)

	body, err := JSONFromPlatforms(platforms)
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

	return checkCreatePlatformsResponse(ret, platforms)
}

/**
 * @brief 删除平台
 */
func (c *Client) DeletePlatforms(platforms []*api.Platform) error {
	fmt.Printf("\ndelete platforms\n")

	url := fmt.Sprintf("http://%v/naming/%v/platforms/delete", c.Address, c.Version)

	body, err := JSONFromPlatforms(platforms)
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
 * @brief 更新平台
 */
func (c *Client) UpdatePlatforms(platforms []*api.Platform) error {
	fmt.Printf("\nupdate platforms\n")

	url := fmt.Sprintf("http://%v/naming/%v/platforms", c.Address, c.Version)

	body, err := JSONFromPlatforms(platforms)
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
 * @brief 查询平台
 */
func (c *Client) GetPlatforms(platforms []*api.Platform) error {
	fmt.Printf("\nget platforms\n")

	url := fmt.Sprintf("http://%v/naming/%v/platforms", c.Address, c.Version)

	params := map[string][]interface{}{
		"name": {platforms[0].GetName().GetValue()},
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
		return fmt.Errorf("invalid batch code: %v", ret.GetCode().GetValue())
	}

	platformsSize := len(platforms)

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(platformsSize) {
		return fmt.Errorf("invalid batch amount: %d, expect amount is %d",
			ret.GetAmount().GetValue(), platformsSize)
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(platformsSize) {
		return fmt.Errorf("invalid batch size: %d, expect size is %d",
			ret.GetSize().GetValue(), platformsSize)
	}

	collection := make(map[string]*api.Platform)
	for _, platform := range platforms {
		collection[platform.GetId().GetValue()] = platform
	}

	items := ret.GetPlatforms()
	if items == nil || len(items) != platformsSize {
		return errors.New("invalid batch platforms")
	}

	for _, item := range items {
		if correctItem, ok := collection[item.GetId().GetValue()]; ok {
			if result, err := comparePlatform(correctItem, item); !result {
				return fmt.Errorf("invalid platform. id is %s, err is %s", item.GetId().GetValue(), err.Error())
			}
		} else {
			return fmt.Errorf("platform not found. id is %s", item.GetId().GetValue())
		}
	}
	return nil
}

/**
 * @brief 检查创建平台的回复
 */
func checkCreatePlatformsResponse(ret *api.BatchWriteResponse, platforms []*api.Platform) (
	*api.BatchWriteResponse, error) {
	if ret.GetCode() == nil || ret.GetCode().GetValue() != api.ExecuteSuccess {
		return nil, fmt.Errorf("invalid batch code: %v", ret.GetCode().GetValue())
	}

	platformsSize := len(platforms)
	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(platformsSize) {
		return nil, fmt.Errorf("invalid batch size, expect size is %d, actual size is %d",
			platformsSize, ret.GetSize().GetValue())
	}

	items := ret.GetResponses()
	if items == nil || len(items) != platformsSize {
		return nil, errors.New("invalid batch response")
	}

	for index, item := range items {
		if item.GetCode() == nil || item.GetCode().GetValue() != api.ExecuteSuccess {
			return nil, fmt.Errorf("invalid code: %v", item.GetCode().GetValue())
		}

		platform := item.GetPlatform()
		if platform == nil {
			return nil, errors.New("empty platform")
		}
		if _, err := comparePlatform(platforms[index], platform); err != nil {
			return nil, fmt.Errorf("invalid platform. id is %s, err is %s",
				platform.GetId().GetValue(), err.Error())
		}
	}
	return ret, nil
}

/**
 * @brief 比较平台信息是否相等
 */
func comparePlatform(correctItem *api.Platform, item *api.Platform) (bool, error) {
	switch {
	case correctItem.GetId().GetValue() != item.GetId().GetValue():
		return false, errors.New("error id")
	case correctItem.GetName().GetValue() != item.GetName().GetValue():
		return false, errors.New("error name")
	case correctItem.GetDomain().GetValue() != item.GetDomain().GetValue():
		return false, errors.New("error domain")
	case correctItem.GetQps().GetValue() != item.GetQps().GetValue():
		return false, errors.New("error qps")
	case correctItem.GetOwner().GetValue() != item.GetOwner().GetValue():
		return false, errors.New("error owner")
	case correctItem.GetDepartment().GetValue() != item.GetDepartment().GetValue():
		return false, errors.New("error department")
	case correctItem.GetComment().GetValue() != item.GetComment().GetValue():
		return false, errors.New("error comment")
	}
	return true, nil
}
