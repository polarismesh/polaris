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
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// JSONFromCircuitBreakers marshals a slice of circuit breakers to JSON. 熔断规则数组转JSON
func JSONFromCircuitBreakers(circuitBreakers []*apifault.CircuitBreaker) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, circuitBreaker := range circuitBreakers {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, circuitBreaker)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))
	return buffer, nil
}

// JSONFromConfigReleases marshals a slice of config releases to JSON. 配置发布规则数组转JSON
func JSONFromConfigReleases(configReleases []*apiservice.ConfigRelease) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	buffer.Write([]byte("["))
	for index, configRelease := range configReleases {
		if index > 0 {
			buffer.Write([]byte(",\n"))
		}
		err := m.Marshal(buffer, configRelease)
		if err != nil {
			return nil, err
		}
	}

	buffer.Write([]byte("]"))
	return buffer, nil
}

// CreateCircuitBreakers creates a slice of circuit breakers from JSON. 创建熔断规则
func (c *Client) CreateCircuitBreakers(circuitBreakers []*apifault.CircuitBreaker) (*apiservice.BatchWriteResponse, error) {
	fmt.Printf("\ncreate circuit breakers\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers", c.Address, c.Version)

	body, err := JSONFromCircuitBreakers(circuitBreakers)
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

	return checkCreateCircuitBreakersResponse(ret, circuitBreakers)
}

// CreateCircuitBreakerVersions creates a slice of circuit breakers from JSON. 创建熔断规则版本
func (c *Client) CreateCircuitBreakerVersions(circuitBreakers []*apifault.CircuitBreaker) (*apiservice.BatchWriteResponse, error) {
	fmt.Printf("\ncreate circuit breaker versions\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers/version", c.Address, c.Version)
	body, err := JSONFromCircuitBreakers(circuitBreakers)
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

	return checkCreateCircuitBreakersResponse(ret, circuitBreakers)
}

// UpdateCircuitBreakers 更新熔断规则
func (c *Client) UpdateCircuitBreakers(circuitBreakers []*apifault.CircuitBreaker) error {
	fmt.Printf("\nupdate circuit breakers\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers", c.Address, c.Version)

	body, err := JSONFromCircuitBreakers(circuitBreakers)
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
 * @brief 删除熔断规则
 */
func (c *Client) DeleteCircuitBreakers(circuitBreakers []*apifault.CircuitBreaker) error {
	fmt.Printf("\ndelete circuit breakers\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers/delete", c.Address, c.Version)

	body, err := JSONFromCircuitBreakers(circuitBreakers)
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
 * @brief 发布熔断规则
 */
func (c *Client) ReleaseCircuitBreakers(configReleases []*apiservice.ConfigRelease) error {
	fmt.Printf("\nrelease circuit breakers\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers/release", c.Address, c.Version)

	body, err := JSONFromConfigReleases(configReleases)
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
 * @brief 解绑熔断规则
 */
func (c *Client) UnbindCircuitBreakers(configReleases []*apiservice.ConfigRelease) error {
	fmt.Printf("\nunbind circuit breakers\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers/unbind", c.Address, c.Version)

	body, err := JSONFromConfigReleases(configReleases)
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
 * @brief 根据id和version查询熔断规则
 */
func (c *Client) GetCircuitBreaker(masterCircuitBreaker, circuitBreaker *apifault.CircuitBreaker) error {
	fmt.Printf("\nget circuit breaker by id and version\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreaker", c.Address, c.Version)

	params := map[string][]interface{}{
		"id":      {circuitBreaker.GetId().GetValue()},
		"version": {circuitBreaker.GetVersion().GetValue()},
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

	size := 1

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(size) {
		return errors.New("invalid batch amount")
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(size) {
		return errors.New("invalid batch size")
	}

	item := ret.GetConfigWithServices()
	if item == nil || len(item) != size {
		return errors.New("invalid batch circuit breakers")
	}

	if item[0].GetCircuitBreaker() == nil {
		return errors.New("invalid circuit breakers")
	}

	if result, err := compareCircuitBreaker(circuitBreaker, masterCircuitBreaker, item[0].GetCircuitBreaker()); !result {
		return err
	}

	return nil
}

/**
 * @brief 查询熔断规则的已发布规则及服务
 */
func (c *Client) GetCircuitBreakersRelease(circuitBreaker *apifault.CircuitBreaker, correctService *apiservice.Service) error {
	fmt.Printf("\nget circuit breaker release\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreakers/release", c.Address, c.Version)

	params := map[string][]interface{}{
		"id": {circuitBreaker.GetId().GetValue()},
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

	size := 1

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(size) {
		return fmt.Errorf("invalid batch amount, expect : %d, actual : %d", size, ret.GetAmount().GetValue())
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(size) {
		return errors.New("invalid batch size")
	}

	configWithServices := ret.GetConfigWithServices()
	if configWithServices == nil || len(configWithServices) != size {
		return errors.New("invalid batch circuit breakers")
	}

	if configWithServices[0].GetCircuitBreaker() == nil {
		return errors.New("invalid circuit breakers")
	}

	rule := configWithServices[0].GetCircuitBreaker()

	if circuitBreaker.GetId().GetValue() != rule.GetId().GetValue() ||
		circuitBreaker.GetVersion().GetValue() != rule.GetVersion().GetValue() {
		return errors.New("error circuit breaker id or version")
	}

	if configWithServices[0].GetServices() == nil || configWithServices[0].GetServices()[0] == nil {
		return errors.New("invalid services")
	}

	service := configWithServices[0].GetServices()[0]
	serviceName := service.GetName().GetValue()
	namespaceName := service.GetNamespace().GetValue()

	if serviceName != correctService.GetName().GetValue() ||
		namespaceName != correctService.GetNamespace().GetValue() {
		return errors.New("invalid service name or namespace")
	}

	return nil
}

/**
 * @brief 查询熔断规则所有版本
 */
func (c *Client) GetCircuitBreakerVersions(circuitBreaker *apifault.CircuitBreaker) error {
	fmt.Printf("\nget circuit breaker versions\n")

	url := fmt.Sprintf("http://%v/naming/%v/circuitbreaker/versions", c.Address, c.Version)

	params := map[string][]interface{}{
		"id": {circuitBreaker.GetId().GetValue()},
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

	size := 2

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(size) {
		return errors.New("invalid batch amount")
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(size) {
		return errors.New("invalid batch size")
	}

	configWithServices := ret.GetConfigWithServices()
	if configWithServices == nil || len(configWithServices) != size {
		return errors.New("invalid batch circuit breakers")
	}

	versions := make([]string, 0, size)
	for _, item := range configWithServices {
		cb := item.GetCircuitBreaker()
		if cb.GetId().GetValue() != circuitBreaker.GetId().GetValue() {
			return errors.New("invalid circuit breaker id")
		}
		versions = append(versions, cb.GetVersion().GetValue())
	}

	correctVersions := map[string]bool{
		circuitBreaker.GetVersion().GetValue(): true,
		"master":                               true,
	}

	for _, version := range versions {
		if _, ok := correctVersions[version]; !ok {
			return errors.New("invalid circuit breaker version")
		}
	}

	return nil
}

/**
 * @brief 查询服务绑定的熔断规则
 */
func (c *Client) GetCircuitBreakerByService(service *apiservice.Service, masterCircuitBreaker,
	circuitBreaker *apifault.CircuitBreaker) error {
	fmt.Printf("\nget circuit breaker by service\n")

	url := fmt.Sprintf("http://%v/naming/%v/service/circuitbreaker", c.Address, c.Version)

	params := map[string][]interface{}{
		"service":   {service.GetName().GetValue()},
		"namespace": {service.GetNamespace().GetValue()},
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

	size := 1

	if ret.GetAmount() == nil || ret.GetAmount().GetValue() != uint32(size) {
		return errors.New("invalid batch amount")
	}

	if ret.GetSize() == nil || ret.GetSize().GetValue() != uint32(size) {
		return errors.New("invalid batch size")
	}

	configWithServices := ret.GetConfigWithServices()
	if configWithServices == nil || len(configWithServices) != size {
		return errors.New("invalid batch circuit breakers")
	}

	rule := configWithServices[0].GetCircuitBreaker()
	if rule == nil {
		return errors.New("invalid circuit breaker")
	}

	if result, err := compareCircuitBreaker(circuitBreaker, masterCircuitBreaker, rule); !result {
		return err
	}

	return nil
}

/**
 * @brief 检查创建熔断规则的回复
 */
func checkCreateCircuitBreakersResponse(ret *apiservice.BatchWriteResponse, circuitBreakers []*apifault.CircuitBreaker) (
	*apiservice.BatchWriteResponse, error) {
	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New("invalid batch code")
	case ret.GetSize().GetValue() != uint32(len(circuitBreakers)):
		return nil, errors.New("invalid batch size")
	case len(ret.GetResponses()) != len(circuitBreakers):
		return nil, errors.New("invalid batch response")
	}

	for index, item := range ret.GetResponses() {
		if item.GetCode().GetValue() != api.ExecuteSuccess {
			return nil, errors.New("invalid code")
		}
		circuitBreaker := item.GetCircuitBreaker()
		if circuitBreaker == nil {
			return nil, errors.New("empty circuit breaker")
		}

		if result, err := compareCircuitBreaker(circuitBreakers[index], circuitBreakers[index], circuitBreaker); !result {
			return nil, err
		} else {
			return ret, nil
		}
	}
	return ret, nil
}

/**
 * @brief 比较circuit breaker是否相等
 */
func compareCircuitBreaker(correctItem, correctMaster *apifault.CircuitBreaker, item *apifault.CircuitBreaker) (bool, error) {
	switch {
	case item.GetId() == nil || item.GetId().GetValue() == "":
		return false, errors.New("error id")
	case item.GetVersion() == nil || item.GetVersion().GetValue() == "":
		return false, errors.New("error version")
	case correctMaster.GetName().GetValue() != item.GetName().GetValue():
		return false, errors.New("error name")
	case correctMaster.GetNamespace().GetValue() != item.GetNamespace().GetValue():
		return false, errors.New("error namespace")
	case correctMaster.GetOwners().GetValue() != item.GetOwners().GetValue():
		return false, errors.New("error owners")
	case correctMaster.GetComment().GetValue() != item.GetComment().GetValue():
		return false, errors.New("error comment")
	case correctMaster.GetBusiness().GetValue() != item.GetBusiness().GetValue():
		return false, errors.New("error business")
	case correctMaster.GetDepartment().GetValue() != item.GetDepartment().GetValue():
		return false, errors.New("error department")
	default:
		break
	}

	correctInbounds, err := json.Marshal(correctItem.GetInbounds())
	if err != nil {
		panic(err)
	}
	inbounds, err := json.Marshal(item.GetInbounds())
	if err != nil {
		panic(err)
	}
	if string(correctInbounds) != string(inbounds) {
		return false, errors.New("error inbounds")
	}

	correctOutbounds, err := json.Marshal(correctItem.GetOutbounds())
	if err != nil {
		panic(err)
	}
	outbounds, err := json.Marshal(item.GetOutbounds())
	if err != nil {
		panic(err)
	}
	if string(correctOutbounds) != string(outbounds) {
		return false, errors.New("error inbounds")
	}
	return true, nil
}
