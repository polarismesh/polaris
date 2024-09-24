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

package defaultuser

import (
	"fmt"
	golog "log"

	"github.com/polarismesh/polaris/auth"
	user_auth "github.com/polarismesh/polaris/auth/user/inteceptor/auth"
	"github.com/polarismesh/polaris/auth/user/inteceptor/paramcheck"
)

type ServerProxyFactory func(svr *Server, pre auth.UserServer) (auth.UserServer, error)

var (
	// serverProxyFactories auth.UserServer API 代理工厂
	serverProxyFactories = map[string]ServerProxyFactory{}
)

// RegisterServerProxy .
func RegisterServerProxy(name string, factor ServerProxyFactory) {
	if _, ok := serverProxyFactories[name]; ok {
		golog.Printf("duplicate ServerProxyFactory, name(%s)", name)
		return
	}
	serverProxyFactories[name] = factor
}

func init() {
	_, nextSvr, err := BuildServer()
	if err != nil {
		panic(err)
	}
	_ = auth.RegisterUserServer(nextSvr)
}

func loadInteceptors() {
	RegisterServerProxy("auth", func(svr *Server, pre auth.UserServer) (auth.UserServer, error) {
		return user_auth.NewServer(pre), nil
	})
	RegisterServerProxy("paramcheck", func(svr *Server, pre auth.UserServer) (auth.UserServer, error) {
		return paramcheck.NewServer(pre), nil
	})
}

func BuildServer() (*Server, auth.UserServer, error) {
	loadInteceptors()
	svr := &Server{}
	var nextSvr auth.UserServer
	nextSvr = svr
	// 需要返回包装代理的 DiscoverServer
	order := GetChainOrder()
	for i := range order {
		factory, exist := serverProxyFactories[order[i]]
		if !exist {
			return nil, nil, fmt.Errorf("name(%s) not exist in serverProxyFactories", order[i])
		}

		proxySvr, err := factory(svr, nextSvr)
		if err != nil {
			panic(err)
		}
		nextSvr = proxySvr
	}
	return svr, nextSvr, nil
}

func GetChainOrder() []string {
	return []string{
		"auth",
		"paramcheck",
	}
}
