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
	"mime/multipart"
	"net/http"
	"os"

	"github.com/golang/protobuf/jsonpb"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

/**
 * @brief 实例数组转JSON
 */
func JSONFromConfigGroup(group *apiconfig.ConfigFileGroup) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, group)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func JSONFromConfigFile(file *apiconfig.ConfigFile) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, file)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func JSONFromConfigFileRelease(file *apiconfig.ConfigFileRelease) (*bytes.Buffer, error) {
	m := jsonpb.Marshaler{Indent: " "}

	buffer := bytes.NewBuffer([]byte{})

	err := m.Marshal(buffer, file)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (c *Client) CreateConfigGroup(group *apiconfig.ConfigFileGroup) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) UpdateConfigGroup(group *apiconfig.ConfigFileGroup) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) QueryConfigGroup(group *apiconfig.ConfigFileGroup,
	offset, limit int64) (*apiconfig.ConfigBatchQueryResponse, error) {
	fmt.Printf("\nquery config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups?namespace=%s&group=%s&offset=%d&limit=%d",
		c.Address, c.Version, group.Namespace.GetValue(), group.Name.GetValue(), offset, limit)

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

func (c *Client) DeleteConfigGroup(group *apiconfig.ConfigFileGroup) (*apiconfig.ConfigResponse, error) {
	fmt.Printf("\ndelete config_file_groups\n")

	url := fmt.Sprintf("http://%v/config/%v/configfilegroups?namespace=%s&group=%s",
		c.Address, c.Version, group.Namespace.GetValue(), group.Name.GetValue())

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

func (c *Client) CreateConfigFile(file *apiconfig.ConfigFile) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) UpdateConfigFile(file *apiconfig.ConfigFile) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) DeleteConfigFile(file *apiconfig.ConfigFile) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) ExportConfigFile(req *apiconfig.ConfigFileExportRequest) error {
	fmt.Printf("\nexport config_file\n")

	url := fmt.Sprintf("http://%v/config/%v/configfiles/export", c.Address, c.Version)

	m := jsonpb.Marshaler{Indent: " "}
	body := bytes.NewBuffer([]byte{})
	err := m.Marshal(body, req)
	if err != nil {
		return err
	}

	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errors.New("invalid http code")
	}
	if response.Header.Get("Content-Type") != "application/zip" {
		return errors.New("invalid content type")
	}

	defer response.Body.Close()
	out, err := os.Create("export.zip")
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, response.Body); err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	return nil
}

func (c *Client) ImportConfigFile(namespace, group, conflictHandling string) (*apiconfig.ConfigImportResponse, error) {
	fmt.Printf("\nimport config_file\n")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("namespace", namespace); err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	if err := mw.WriteField("group", group); err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	if err := mw.WriteField("conflict_handling", conflictHandling); err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	filename := "export.zip"
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	defer f.Close()
	fw, _ := mw.CreateFormFile("config", filename)
	if _, err := io.Copy(fw, f); err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	mw.Close()

	url := fmt.Sprintf("http://%v/config/%v/configfiles/import?namespace=%s&group=%s&conflict_handling=%s", c.Address,
		c.Version, namespace, group, conflictHandling)

	request, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	request.Header.Add("Content-Type", mw.FormDataContentType())
	request.Header.Add("Request-Id", "test")
	request.Header.Add("X-Polaris-Token",
		"nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=")

	response, err := c.Worker.Do(request)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	ret, err := GetConfigImportResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	if ret.GetCode().GetValue() != api.ExecuteSuccess {
		return nil, errors.New(ret.GetInfo().GetValue())
	}
	return ret, nil
}

func (c *Client) CreateConfigFileRelease(file *apiconfig.ConfigFileRelease) (*apiconfig.ConfigResponse, error) {
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

func (c *Client) GetAllConfigEncryptAlgorithms() (*apiconfig.ConfigEncryptAlgorithmResponse, error) {
	fmt.Printf("\nquery config encrypt algorithm\n")
	url := fmt.Sprintf("http://%v/config/%v/configfiles/encryptalgorithm", c.Address, c.Version)
	response, err := c.SendRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	ret, err := GetConfigEncryptAlgorithmResponse(response)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	if ret.GetCode().GetValue() != api.ExecuteSuccess {
		return nil, errors.New(ret.GetInfo().GetValue())
	}
	return ret, nil
}

func checkCreateConfigResponse(ret *apiconfig.ConfigResponse) (
	*apiconfig.ConfigResponse, error) {

	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New(ret.GetInfo().GetValue())
	}

	return ret, nil
}

func checkQueryConfigResponse(ret *apiconfig.ConfigBatchQueryResponse) (
	*apiconfig.ConfigBatchQueryResponse, error) {

	switch {
	case ret.GetCode().GetValue() != api.ExecuteSuccess:
		return nil, errors.New(ret.GetInfo().GetValue())
	}

	return ret, nil
}
