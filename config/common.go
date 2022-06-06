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
 * CONDITIONS OF ANY KIND, either express or Serveried. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"context"
	"errors"
	api "github.com/polarismesh/polaris-server/common/api/v1"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	ContextTxKey = utils.StringContext("Config-Tx")
)

// StartTxAndSetToContext 开启一个事务，并放入到上下文里
func (cs *Server) StartTxAndSetToContext(ctx context.Context) (store.Tx, context.Context, error) {
	tx, err := cs.storage.StartTx()
	return tx, context.WithValue(ctx, ContextTxKey, tx), err
}

// getTx 从上下文里获取事务对象
func (cs *Server) getTx(ctx context.Context) store.Tx {
	tx := ctx.Value(ContextTxKey)
	if tx == nil {
		return nil
	}
	return tx.(store.Tx)
}

func (cs *Server) checkNamespaceExisted(namespaceName string) bool {
	namespace, _ := cs.storage.GetNamespace(namespaceName)
	return namespace != nil
}

func (cs *Server) createNamespaceIfAbsent(namespaceName, operator, requestId string) error {
	namespace, err := cs.storage.GetNamespace(namespaceName)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get namespace error.", zap.Error(err))
		return err
	}
	if namespace != nil {
		return nil
	}

	namespace = &model.Namespace{
		Name:    namespaceName,
		Token:   utils.NewUUID(),
		Owner:   operator,
		Comment: "auto created by config module",
	}

	if err := cs.storage.AddNamespace(namespace); err != nil {
		log.ConfigScope().Error("[Config][Service] create namespace error.",
			zap.String("namespace", namespaceName),
			zap.String("requestId", requestId),
			zap.Error(err))
		return err
	}

	return nil
}

func convertToErrCode(err error) uint32 {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return api.TokenNotExisted
	}
	if errors.Is(err, model.ErrorTokenDisabled) {
		return api.TokenDisabled
	}
	return api.NotAllowedAccess
}
