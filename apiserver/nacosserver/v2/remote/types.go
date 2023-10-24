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

package remote

import (
	"context"
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/types/known/anypb"

	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
)

var (
	ErrorNoSuchPayloadType      = errors.New("not such payload type")
	ErrorInvalidRequestBodyType = errors.New("invalid request body type")
)

type (
	// RequestHandler
	RequestHandler func(context.Context, nacospb.BaseRequest, nacospb.RequestMeta) (nacospb.BaseResponse, error)
	// RequestHandlerWarrper
	RequestHandlerWarrper struct {
		Handler        RequestHandler
		PayloadBuilder func() nacospb.CustomerPayload
	}
)

// MarshalPayload .
func MarshalPayload(valu interface{}) (*nacospb.Payload, error) {
	switch resp := valu.(type) {
	case nacospb.BaseResponse:
		data, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		payload := &nacospb.Payload{
			Metadata: &nacospb.Metadata{
				Type: resp.GetResponseType(),
			},
			Body: &anypb.Any{
				Value: data,
			},
		}
		return payload, nil
	case nacospb.BaseRequest:
		data, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		payload := &nacospb.Payload{
			Metadata: &nacospb.Metadata{
				Type: resp.GetRequestType(),
			},
			Body: &anypb.Any{
				Value: data,
			},
		}
		return payload, nil
	default:
		return nil, errors.New("value no pb.BaseResponse or pb.BaseRequest")
	}
}
