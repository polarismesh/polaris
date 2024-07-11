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

package policy_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris/auth"
	authmock "github.com/polarismesh/polaris/auth/mock"
	"github.com/polarismesh/polaris/auth/policy"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	storemock "github.com/polarismesh/polaris/store/mock"
	"github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Test_AfterResourceOperation(t *testing.T) {
	svr := &policy.Server{}

	t.Run("not_need_auth", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(false).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		err := svr.AfterResourceOperation(model.NewAcquireContext())
		assert.NoError(t, err)
	})

	t.Run("read_op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		err := svr.AfterResourceOperation(model.NewAcquireContext(
			model.WithOperation(model.Read),
		))
		assert.NoError(t, err)
	})

	t.Run("from_client_not_auth", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(false).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(true).AnyTimes()

		err := svr.AfterResourceOperation(model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromClient(),
		))
		assert.NoError(t, err)
	})

	t.Run("from_console_not_auth", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		err := svr.AfterResourceOperation(model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromConsole(),
		))
		assert.NoError(t, err)
	})

	t.Run("not_token_detial", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		err := svr.AfterResourceOperation(model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromClient(),
		))
		assert.NoError(t, err)
	})

	t.Run("invalid_token_detial", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		ctx := model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromClient(),
		)
		ctx.SetAttachment(model.TokenDetailInfoKey, map[string]string{})
		err := svr.AfterResourceOperation(ctx)
		assert.NoError(t, err)
	})

	t.Run("empty_token_detial", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(false).AnyTimes()

		ctx := model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromClient(),
		)
		t.Run("origin_empty", func(t *testing.T) {
			ctx.SetAttachment(model.TokenDetailInfoKey, auth.OperatorInfo{
				Origin: "",
			})
			err := svr.AfterResourceOperation(ctx)
			assert.NoError(t, err)
		})

		t.Run("is_anonymous", func(t *testing.T) {
			ctx.SetAttachment(model.TokenDetailInfoKey, auth.OperatorInfo{
				Origin:    "123",
				Anonymous: true,
			})
			err := svr.AfterResourceOperation(ctx)
			assert.NoError(t, err)
		})
	})

	t.Run("change_principal_policy", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockChecker := authmock.NewMockAuthChecker(ctrl)
		svr.MockAuthChecker(mockChecker)
		mockChecker.EXPECT().IsOpenClientAuth().Return(true).AnyTimes()
		mockChecker.EXPECT().IsOpenConsoleAuth().Return(true).AnyTimes()

		ctx := model.NewAcquireContext(
			model.WithOperation(model.Create),
			model.WithFromClient(),
		)

		ownerId := "mock_auth_owner"
		curUserId := "123"

		t.Run("user", func(t *testing.T) {
			ctx.SetAttachment(model.TokenDetailInfoKey, auth.OperatorInfo{
				Origin:      curUserId,
				OperatorID:  curUserId,
				OwnerID:     ownerId,
				Role:        model.OwnerUserRole,
				IsUserToken: true,
			})

			initMockAcquireContext(ctx)

			t.Run("not_found_user", func(t *testing.T) {
				userSvr := authmock.NewMockUserServer(ctrl)
				mockHelper := authmock.NewMockUserHelper(ctrl)

				userSvr.EXPECT().GetUserHelper().Return(mockHelper)
				mockHelper.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(nil)

				svr.MockUserServer(userSvr)

				err := svr.AfterResourceOperation(ctx)
				assert.Error(t, err)
				assert.Equal(t, "not found target user", err.Error())
			})

			t.Run("found_user", func(t *testing.T) {
				userSvr := authmock.NewMockUserServer(ctrl)
				mockHelper := authmock.NewMockUserHelper(ctrl)

				userSvr.EXPECT().GetUserHelper().Return(mockHelper).AnyTimes()
				mockHelper.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&security.User{
					Id:    wrapperspb.String(curUserId),
					Owner: wrapperspb.String(ownerId),
				}).AnyTimes()

				svr.MockUserServer(userSvr)

				t.Run("store_has_err", func(t *testing.T) {
					sctrl := gomock.NewController(t)
					defer sctrl.Finish()
					mockStore := storemock.NewMockStore(sctrl)

					mockStore.EXPECT().GetDefaultStrategyDetailByPrincipal(gomock.Any(), gomock.Any()).Return(nil, errors.New("mock_err"))
					svr.MockStore(mockStore)

					err := svr.AfterResourceOperation(ctx)
					assert.Error(t, err)
					assert.Equal(t, "mock_err", err.Error())
				})

				t.Run("not_found_default_policy", func(t *testing.T) {
					sctrl := gomock.NewController(t)
					defer sctrl.Finish()
					mockStore := storemock.NewMockStore(sctrl)

					mockStore.EXPECT().GetDefaultStrategyDetailByPrincipal(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
					svr.MockStore(mockStore)

					err := svr.AfterResourceOperation(ctx)
					assert.Error(t, err)
					assert.Equal(t, "not found default strategy rule", err.Error())
				})

				t.Run("not_op_resource", func(t *testing.T) {
					sctrl := gomock.NewController(t)
					defer sctrl.Finish()
					mockStore := storemock.NewMockStore(sctrl)

					mockStore.EXPECT().GetDefaultStrategyDetailByPrincipal(gomock.Any(), gomock.Any()).Return(&model.StrategyDetail{}, nil)
					svr.MockStore(mockStore)

					err := svr.AfterResourceOperation(ctx)
					assert.NoError(t, err)
				})

				t.Run("invalid_op_resource", func(t *testing.T) {
					sctrl := gomock.NewController(t)
					defer sctrl.Finish()
					mockStore := storemock.NewMockStore(sctrl)

					mockStore.EXPECT().GetDefaultStrategyDetailByPrincipal(gomock.Any(), gomock.Any()).Return(&model.StrategyDetail{}, nil)
					svr.MockStore(mockStore)

					ctx.SetAttachment(model.ResourceAttachmentKey, map[string]interface{}{})

					err := svr.AfterResourceOperation(ctx)
					assert.NoError(t, err)
				})

				t.Run("delete_resource", func(t *testing.T) {
					delCtx := model.NewAcquireContext(
						model.WithOperation(model.Delete),
						model.WithFromClient(),
					)
					delCtx.SetAttachment(model.TokenDetailInfoKey, auth.OperatorInfo{
						Origin:      curUserId,
						OperatorID:  curUserId,
						OwnerID:     ownerId,
						Role:        model.OwnerUserRole,
						IsUserToken: true,
					})

					initMockAcquireContext(delCtx)

					sctrl := gomock.NewController(t)
					defer sctrl.Finish()
					mockStore := storemock.NewMockStore(sctrl)
					mockStore.EXPECT().RemoveStrategyResources(gomock.Any()).DoAndReturn(func(args interface{}) error {
						resources := args.([]model.StrategyResource)
						for i := range resources {
							assert.True(t, resources[i].StrategyID == "", utils.MustJson(resources[i]))
						}
						return nil
					}).Times(1)

					svr.MockStore(mockStore)
					err := svr.AfterResourceOperation(delCtx)
					assert.NoError(t, err)
				})
			})
		})
	})
}

func initMockAcquireContext(ctx *model.AcquireContext) {
	ctx.SetAttachment(model.LinkUsersKey, []string{})
	ctx.SetAttachment(model.LinkGroupsKey, []string{})
	ctx.SetAttachment(model.RemoveLinkUsersKey, []string{})
	ctx.SetAttachment(model.RemoveLinkGroupsKey, []string{})
}
