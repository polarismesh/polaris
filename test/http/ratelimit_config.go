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
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

/**
 * @brief 限流规则数组转JSON
 */
func JSONFromRateLimits(rateLimits []*apitraffic.Rule) (*bytes.Buffer, error) {
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
func (c *Client) CreateRateLimits(rateLimits []*apitraffic.Rule) (*apiservice.BatchWriteResponse, error) {
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
		return ret, err
	}

	return checkCreateRateLimitsResponse(ret, rateLimits)
}

/**
 * @brief 删除限流规则
 */
func (c *Client) DeleteRateLimits(rateLimits []*apitraffic.Rule) error {
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
func (c *Client) UpdateRateLimits(rateLimits []*apitraffic.Rule) error {
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

// EnableRateLimits 启用限流规则
func (c *Client) EnableRateLimits(rateLimits []*apitraffic.Rule) error {
	fmt.Printf("\nenable rate limits\n")

	url := fmt.Sprintf("http://%v/naming/%v/ratelimits/enable", c.Address, c.Version)

	rateLimitsEnable := make([]*apitraffic.Rule, 0, len(rateLimits))
	for _, rateLimit := range rateLimits {
		rateLimitsEnable = append(rateLimitsEnable, &apitraffic.Rule{
			Id:      rateLimit.GetId(),
			Disable: &wrappers.BoolValue{Value: true},
		})
	}
	body, err := JSONFromRateLimits(rateLimitsEnable)
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
func (c *Client) GetRateLimits(rateLimits []*apitraffic.Rule) error {
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

	collection := make(map[string]*apitraffic.Rule)
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
func checkCreateRateLimitsResponse(ret *apiservice.BatchWriteResponse, rateLimits []*apitraffic.Rule) (
	*apiservice.BatchWriteResponse, error) {
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
func compareRateLimit(correctItem *apitraffic.Rule, item *apitraffic.Rule) (bool, error) {
	switch {
	case (correctItem.GetId().GetValue()) != "" && (correctItem.GetId().GetValue() != item.GetId().GetValue()):
		return false, fmt.Errorf(
			"invalid id, expect %s, actual %s", correctItem.GetId().GetValue(), item.GetId().GetValue())
	case correctItem.GetService().GetValue() != item.GetService().GetValue():
		return false, fmt.Errorf("error service, expect %s, actual %s",
			correctItem.GetService().GetValue(), item.GetService().GetValue())
	case correctItem.GetNamespace().GetValue() != item.GetNamespace().GetValue():
		return false, fmt.Errorf("error namespace, expect %s, actual %s",
			correctItem.GetNamespace().GetValue(), item.GetNamespace().GetValue())
	case correctItem.GetPriority().GetValue() != item.GetPriority().GetValue():
		return false, fmt.Errorf("invalid priority, expect %v, actual %v",
			correctItem.GetPriority().GetValue(), item.GetPriority().GetValue())
	case correctItem.GetResource() != item.GetResource():
		return false, fmt.Errorf("invalid resource, expect %v, actual %v",
			correctItem.GetResource(), item.GetResource())
	case correctItem.GetType() != item.GetType():
		return false, fmt.Errorf("error type, exepct %v, actual %v", correctItem.GetType(), item.GetType())
	case correctItem.GetAction().GetValue() != item.GetAction().GetValue():
		return false, fmt.Errorf("error action, expect %v, actual %v",
			correctItem.GetAction().GetValue(), item.GetAction().GetValue())
	case correctItem.GetDisable().GetValue() != item.GetDisable().GetValue():
		return false, fmt.Errorf("error disable, expect %v, actual %v",
			correctItem.GetDisable().GetValue(), item.GetDisable().GetValue())
	case correctItem.GetRegexCombine().GetValue() != item.GetRegexCombine().GetValue():
		return false, fmt.Errorf("error regex combine, expect %v, actual %v",
			correctItem.GetRegexCombine().GetValue(), item.GetRegexCombine().GetValue())
	case correctItem.GetAmountMode() != item.GetAmountMode():
		return false, fmt.Errorf("error amount mode, expect %v, actual %v",
			correctItem.GetAmountMode(), item.GetAmountMode())
	case correctItem.GetFailover() != item.GetFailover():
		return false, fmt.Errorf(
			"error fail over, expect %v, actual %v", correctItem.GetFailover(), item.GetFailover())
	default:
		break
	}

	if equal, err := checkField(correctItem.GetArguments(), item.GetArguments(), "arguments"); !equal {
		return equal, err
	}

	if equal, err := checkField(correctItem.GetAmounts(), item.GetAmounts(), "amounts"); !equal {
		return equal, err
	}

	if equal, err := checkField(correctItem.GetAdjuster(), item.GetAdjuster(), "adjuster"); !equal {
		return equal, err
	}

	return checkField(correctItem.GetName(), item.GetName(), "cluster")
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
		return false, fmt.Errorf("error %s, expect %s ,actual %s", name, expect, actual)
	}
	return true, nil
}
