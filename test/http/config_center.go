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

	"github.com/golang/protobuf/jsonpb"

	api "github.com/polarismesh/polaris/common/api/v1"
)

/**
 * @brief 实例数组转JSON
 */
func JSONFromConfigGroup(group *api.ConfigFileGroup) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, group)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func JSONFromConfigFile(file *api.ConfigFile) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, file)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func JSONFromConfigFileRelease(file *api.ConfigFileRelease) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, file)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (c *Client) CreateConfigGroup(group *api.ConfigFileGroup) (*api.ConfigResponse, error) {
	fmt.Printf("\ncreate config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups", c.Address, c.Version)

	body, err := JSONFromConfigGroup(group)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) UpdateConfigGroup(group *api.ConfigFileGroup) (*api.ConfigResponse, error) {
	fmt.Printf("\nupdate config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups", c.Address, c.Version)

	body, err := JSONFromConfigGroup(group)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("PUT", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) QueryConfigGroup(group *api.ConfigFileGroup, offset, limit int64) (*api.ConfigBatchQueryResponse, error) {
	fmt.Printf("\nquery config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups?namespace=%s&group=%s&fileName=%s&offset=%d&limit=%d",
		c.Address, c.Version, group.Namespace.GetValue(), group.Name.GetValue(), "", offset, limit)

	body, err := JSONFromConfigGroup(group)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("GET", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigQueryResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkQueryConfigResponse(ret)
}

func (c *Client) DeleteConfigGroup(group *api.ConfigFileGroup) (*api.ConfigResponse, error) {
	fmt.Printf("\ndelete config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups?namespace=%s&group=%s", c.Address, c.Version, group.Namespace.GetValue(), group.Name.GetValue())

	body, err := JSONFromConfigGroup(group)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("DELETE", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) CreateConfigFile(file *api.ConfigFile) (*api.ConfigResponse, error) {
	fmt.Printf("\ncreate config_file\n")

	url := fmt.Sprintf("http://%v/config/%v/configfiles", c.Address, c.Version)

	body, err := JSONFromConfigFile(file)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) UpdateConfigFile(file *api.ConfigFile) (*api.ConfigResponse, error) {
	fmt.Printf("\nupdate config_file\n")

	url := fmt.Sprintf("http://%v/config/%v/configfiles", c.Address, c.Version)

	body, err := JSONFromConfigFile(file)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("PUT", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) DeleteConfigFile(file *api.ConfigFile) (*api.ConfigResponse, error) {
	fmt.Printf("\ndelete config_file\n")

	url := fmt.Sprintf("http://%v/config/%v/configfiles?namespace=%s&group=%s&name=%s", c.Address, c.Version,
		file.Namespace.GetValue(), file.Group.GetValue(), file.Name.GetValue())

	body, err := JSONFromConfigFile(file)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("DELETE", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func (c *Client) CreateConfigFileRelease(file *api.ConfigFileRelease) (*api.ConfigResponse, error) {
	fmt.Printf("\ncreate config_file_release\n")

	url := fmt.Sprintf("http://%v/config/%v/configfiles/release", c.Address, c.Version)

	body, err := JSONFromConfigFileRelease(file)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	return checkCreateConfigResponse(ret)
}

func checkCreateConfigResponse(ret *api.ConfigResponse) (
	*api.ConfigResponse, error) {

	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New(ret.GetInfo().GetValue())
	}

	return ret, nil
}

func checkQueryConfigResponse(ret *api.ConfigBatchQueryResponse) (
	*api.ConfigBatchQueryResponse, error) {

	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New(ret.GetInfo().GetValue())
	}

	return ret, nil
}
