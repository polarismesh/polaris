//go:build integration
// +build integration

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

package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/protobuf/jsonpb"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	phttp "github.com/polarismesh/polaris-server/test/http"
)

func TestUser(t *testing.T) {

}

func login(t *testing.T) (*api.Response, error) {
	c := phttp.NewClient(httpserverAddress, httpserverVersion)
	url := fmt.Sprintf("http://%v/core/%v/user/login", c.Address, c.Version)
	body, err := phttp.JSONFromLoginRequest(&api.LoginRequest{
		// 北极星内置的用户
		Name:     utils.NewStringValue("polaris"),
		Password: utils.NewStringValue("polaris"),
	})
	if err != nil {
		t.Fatal(err)
		return nil, err
	}
	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		t.Fatalf("request user login occurs error: %s", err.Error())
		return nil, err
	}
	ret := &api.Response{}
	if ierr := jsonpb.Unmarshal(response.Body, ret); ierr != nil {
		t.Fatalf("parse user login resp occurs error: %s", err.Error())
		return nil, ierr
	}
	return ret, nil
}

func TestUserLogin(t *testing.T) {
	t.Log("test user login")
	ret, err := login(t)
	if err != nil {
		t.Errorf("user: polaris should req login success, err: %s", err.Error())
		return
	}
	if ret.Code.GetValue() != api.ExecuteSuccess {
		t.Errorf("user: polaris should login success")
		return
	}
	type jwtClaims struct {
		UserID string
		jwt.RegisteredClaims
	}
	token, err := jwt.ParseWithClaims(ret.GetLoginResponse().Token.GetValue(), &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("polarismesh@2021"), nil
	})
	if err != nil {
		t.Errorf("user login resp parse jwt fail: %s", err.Error())
		return
	}
	if claims, ok := token.Claims.(*jwtClaims); !ok || claims.UserID == "" {
		t.Errorf("user login successed token parsed must have user id")
	} else {
		t.Logf("user login parsed user is is %s", claims.UserID)
	}
}

func TestUserRenewalAuthToken(t *testing.T) {
	var request *http.Request
	var err error
	ret, err := login(t)
	if err != nil {
		t.Errorf("user: polaris should req login success, err: %s", err.Error())
		return
	}
	oldToken := ret.LoginResponse.Token.GetValue()
	c := phttp.NewClient(httpserverAddress, httpserverVersion)
	url := fmt.Sprintf("http://%v/core/%v/user/token/renewal", c.Address, c.Version)
	request, err = http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Errorf("renewal token should success, err: %s", err.Error())
		return
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Request-Id", "test")
	request.Header.Add("X-Polaris-Token", oldToken)
	time.Sleep(time.Second)
	response, err := c.Worker.Do(request)
	if err != nil {
		t.Errorf("renewal token should success, err: %s", err.Error())
		return
	}

	ret = &api.Response{}
	if ierr := jsonpb.Unmarshal(response.Body, ret); ierr != nil {
		t.Fatalf("parse user renewal token resp occurs error: %s", ierr.Error())
		return
	}

	if ret.Code.GetValue() != api.ExecuteSuccess || ret.GetLoginResponse() == nil {
		t.Errorf("renewal token should resp success")
		return
	}
	if ret.GetLoginResponse().Token.GetValue() == oldToken {
		t.Errorf("renewal token should not same to old token")
	}

}
