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
	"encoding/json"
	"errors"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/golang/protobuf/jsonpb"
	"io"
)

/**
 * @brief 限流规则数组转JSON
 */
func JSONFromRateLimits(rateLimits []*api.Rule) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, rateLimit := range rateLimits {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, rateLimit)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))
	return buffer, nil
}

/**
 * @brief 创建限流规则
 */
func (c *Client) CreateRateLimits(rateLimits []*api.Rule) (*api.BatchWriteResponse, error) {
	fmt.Printf("\ncreate rate limits\n")

	url := fmt.Sprintf("http://%v/naming/%v/ratelimits", c.Address, c.Version)

	body, err := JSONFromRateLimits(rateLimits)
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

	return checkCreateRateLimitsResponse(ret, rateLimits)
}

/**
 * @brief 删除限流规则
 */
func (c *Client) DeleteRateLimits(rateLimits []*api.Rule) error {
	fmt.Printf("\ndelete rate limits\n")

	url := fmt.Sprintf("http://%v/naming/%v/ratelimits/delete", c.Address, c.Version)

	body, err := JSONFromRateLimits(rateLimits)
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
 * @brief 更新限流规则
 */
func (c *Client) UpdateRateLimits(rateLimits []*api.Rule) error {
	fmt.Printf("\nupdate rate limits\n")

	url := fmt.Sprintf("http://%v/naming/%v/ratelimits", c.Address, c.Version)

	body, err := JSONFromRateLimits(rateLimits)
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
 * @brief 查询限流规则
 */
func (c *Client) GetRateLimits(rateLimits []*api.Rule) error {
	fmt.Printf("\nget rate limits\n")

	url := fmt.Sprintf("http://%v/naming/%v/ratelimits", c.Address, c.Version)

	params := map[string][]interface{}{
		"namespace": {rateLimits[0].GetNamespace().GetValue()},
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

	rateLimitsSize := len(rateLimits)

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(rateLimitsSize) {
		return errors.New("invalid batch amount")
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(rateLimitsSize) {
		return errors.New("invalid batch size")
	}

	collection := make(map[string]*api.Rule)
	for _, rateLimit := range rateLimits {
		collection[rateLimit.GetService().GetValue()] = rateLimit
	}

	items := ret.GetRateLimits()
	if items == nil || len(items) != rateLimitsSize {
		return errors.New("invalid batch rate limits")
	}

	for _, item := range items {
		if correctItem, ok := collection[item.GetService().GetValue()]; ok {
			if result, err := compareRateLimit(correctItem, item); !result {
				return fmt.Errorf("invalid rate limit. namespace is %v, service is %v, err is %s",
					item.GetNamespace().GetValue(), item.GetService().GetValue(), err.Error())
			}
		} else {
			return fmt.Errorf("rate limit not found. namespace is %v, service is %v",
				item.GetNamespace().GetValue(), item.GetService().GetValue())
		}
	}
	return nil
}

/**
 * @brief 检查创建限流规则的回复
 */
func checkCreateRateLimitsResponse(ret *api.BatchWriteResponse, rateLimits []*api.Rule) (
	*api.BatchWriteResponse, error) {
	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New("invalid batch code")
	case ret.GetSize().GetValue() != uint32(len(rateLimits)):
		return nil, errors.New("invalid batch size")
	case len(ret.GetResponses()) != len(rateLimits):
		return nil, errors.New("invalid batch response")
	}

	for index, item := range ret.GetResponses() {
		if item.GetCode().GetValue() != api.ExecuteSuccess {
			return nil, errors.New("invalid code")
		}
		rateLimit := item.GetRateLimit()
		if rateLimit == nil {
			return nil, errors.New("empty rate limit")
		}
		if result, err := compareRateLimit(rateLimits[index], rateLimit); !result {
			return nil, err
		}
	}
	return ret, nil
}

/**
 * @brief 比较rate limit是否相等
 */
func compareRateLimit(correctItem *api.Rule, item *api.Rule) (bool, error) {
	switch {
	case (correctItem.GetId().GetValue()) != "" && (correctItem.GetId().GetValue() != item.GetId().GetValue()):
		return false, errors.New("invalid id")
	case correctItem.GetService().GetValue() != item.GetService().GetValue():
		return false, errors.New("error service")
	case correctItem.GetNamespace().GetValue() != item.GetNamespace().GetValue():
		return false, errors.New("error namespace")
	case correctItem.GetPriority().GetValue() != item.GetPriority().GetValue():
		return false, errors.New("invalid priority")
	case correctItem.GetResource() != item.GetResource():
		return false, errors.New("invalid resource")
	case correctItem.GetType() != item.GetType():
		return false, errors.New("error type")
	case correctItem.GetAction().GetValue() != item.GetAction().GetValue():
		return false, errors.New("error action")
	case correctItem.GetDisable().GetValue() != item.GetDisable().GetValue():
		return false, errors.New("error disable")
	case correctItem.GetRegexCombine().GetValue() != item.GetRegexCombine().GetValue():
		return false, errors.New("error regex combine")
	case correctItem.GetAmountMode() != item.GetAmountMode():
		return false, errors.New("error amount mode")
	case correctItem.GetFailover() != item.GetFailover():
		return false, errors.New("error fail over")
	default:
		break
	}

	if equal, err := checkField(correctItem.GetSubset(), item.GetSubset(), "subset"); !equal {
		return equal, err
	}
	if equal, err := checkField(correctItem.GetLabels(), item.GetLabels(), "labels"); !equal {
		return equal, err
	}

	if equal, err := checkField(correctItem.GetAmounts(), item.GetAmounts(), "amounts"); !equal {
		return equal, err
	}

	if equal, err := checkField(correctItem.GetReport(), item.GetReport(), "report"); !equal {
		return equal, err
	}

	if equal, err := checkField(correctItem.GetAdjuster(), item.GetAdjuster(), "adjuster"); !equal {
		return equal, err
	}

	return checkField(correctItem.GetCluster(), item.GetCluster(), "cluster")
}

/**
 * @brief 检查字段是否一致
 */
func checkField(correctItem, actualItem interface{}, name string) (bool, error) {
	expect, err := json.Marshal(correctItem)
	if err != nil {
		panic(err)
	}
	actual, err := json.Marshal(actualItem)
	if err != nil {
		panic(err)
	}

	if string(expect) != string(actual) {
		return false, fmt.Errorf("error %s", name)
	}
	return true, nil
}
