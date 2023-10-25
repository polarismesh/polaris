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

package model

type NacosErrorCode struct {
	Code int32
	Desc string
}

var (
	/**
	 *  success.
	 */
	ErrorCode_Success = NacosErrorCode{Code: 0, Desc: "success"}

	/**
	 *  parameter missing.
	 */
	ErrorCode_ParameterMissing = NacosErrorCode{Code: 10000, Desc: "parameter missing"}

	/**
	 *  access denied.
	 */
	ErrorCode_AccessDenied = NacosErrorCode{Code: 10001, Desc: "access denied"}

	/**
	 *  data access error.
	 */
	ErrorCode_DataAccessError = NacosErrorCode{Code: 10002, Desc: "data access error"}

	/**
	 *  'tenant' parameter error.
	 */
	ErrorCode_TenantParameterError = NacosErrorCode{Code: 20001, Desc: "'tenant' parameter error"}

	/**
	 *  parameter validate error.
	 */
	ErrorCode_ParameterValidateError = NacosErrorCode{Code: 20002, Desc: "parameter validate error"}

	/**
	 *  MediaType Error.
	 */
	ErrorCode_MediaTypeError = NacosErrorCode{Code: 20003, Desc: "MediaType Error"}

	/**
	 *  resource not found.
	 */
	ErrorCode_ResourceNotFound = NacosErrorCode{Code: 20004, Desc: "resource not found"}

	/**
	 *  resource conflict.
	 */
	ErrorCode_ResourceConflict = NacosErrorCode{Code: 20005, Desc: "resource conflict"}

	/**
	 *  config listener is null.
	 */
	ErrorCode_ConfigListenerIsNull = NacosErrorCode{Code: 20006, Desc: "config listener is null"}

	/**
	 *  config listener error.
	 */
	ErrorCode_ConfigListenerError = NacosErrorCode{Code: 20007, Desc: "config listener error"}

	/**
	 *  invalid dataId.
	 */
	ErrorCode_InvalidDataID = NacosErrorCode{Code: 20008, Desc: "invalid dataId"}

	/**
	 *  parameter mismatch.
	 */
	ErrorCode_ParameterMismatch = NacosErrorCode{Code: 20009, Desc: "parameter mismatch"}

	/**
	 *  service name error.
	 */
	ErrorCode_ServiceNameError = NacosErrorCode{Code: 21000, Desc: "service name error"}

	/**
	 *  weight error.
	 */
	ErrorCode_WeightError = NacosErrorCode{Code: 21001, Desc: "weight error"}

	/**
	 *  instance metadata error.
	 */
	ErrorCode_InstanceMetadataError = NacosErrorCode{Code: 21002, Desc: "instance metadata error"}

	/**
	 *  instance not found.
	 */
	ErrorCode_InstanceNotFound = NacosErrorCode{Code: 21003, Desc: "instance not found"}

	/**
	 *  instance error.
	 */
	ErrorCode_InstanceError = NacosErrorCode{Code: 21004, Desc: "instance error"}

	/**
	 *  service metadata error.
	 */
	ErrorCode_ServiceMetadataError = NacosErrorCode{Code: 21005, Desc: "service metadata error"}

	/**
	 *  selector error.
	 */
	ErrorCode_SelectorError = NacosErrorCode{Code: 21006, Desc: "selector error"}

	/**
	 *  service already exist.
	 */
	ErrorCode_ServiceAlreadyExist = NacosErrorCode{Code: 21007, Desc: "service already exist"}

	/**
	 *  service not exist.
	 */
	ErrorCode_ServiceNotExist = NacosErrorCode{Code: 21008, Desc: "service not exist"}

	/**
	 *  service delete failure.
	 */
	ErrorCode_ServiceDeleteFailure = NacosErrorCode{Code: 21009, Desc: "service delete failure"}

	/**
	 *  healthy param miss.
	 */
	ErrorCode_HealthyParamMiss = NacosErrorCode{Code: 21010, Desc: "healthy param miss"}

	/**
	 *  health check still running.
	 */
	ErrorCode_HealthCheckStillRuning = NacosErrorCode{Code: 21011, Desc: "health check still running"}

	/**
	 *  illegal namespace.
	 */
	ErrorCode_IllegalNamespace = NacosErrorCode{Code: 22000, Desc: "illegal namespace"}

	/**
	 *  namespace not exist.
	 */
	ErrorCode_NamespaceNotExist = NacosErrorCode{Code: 22001, Desc: "namespace not exist"}

	/**
	 *  namespace already exist.
	 */
	ErrorCode_NamespaceAlreadyExist = NacosErrorCode{Code: 22002, Desc: "namespace already exist"}

	/**
	 *  illegal state.
	 */
	ErrorCode_IllegalState = NacosErrorCode{Code: 23000, Desc: "illegal state"}

	/**
	 *  node info error.
	 */
	ErrorCode_NodeInfoError = NacosErrorCode{Code: 23001, Desc: "node info error"}

	/**
	 *  node down failure.
	 */
	ErrorCode_NodeDownFailure = NacosErrorCode{Code: 23001, Desc: "node down failure"}

	/**
	 *  server error.
	 */
	ErrorCode_ServerError = NacosErrorCode{Code: 30000, Desc: "server error"}
)

type ExceptionCode int32

const (
	/**
	 * invalid param（参数错误）.
	 */
	ExceptionCode_ClientInvalidParam ExceptionCode = -400

	/**
	 * client disconnect.
	 */
	ExceptionCode_ClientDisconnect ExceptionCode = -401

	/**
	 * over client threshold（超过client端的限流阈值）.
	 */
	ExceptionCode_ClientOverThreshold ExceptionCode = -503

	/*
	 * server error code.
	 * 400 403 throw exception to user
	 * 500 502 503 change ip and retry
	 */

	/**
	 * invalid param（参数错误）.
	 */
	ExceptionCode_InvalidParam ExceptionCode = 400

	/**
	 * no right（鉴权失败）.
	 */
	ExceptionCode_NoRight ExceptionCode = 403

	/**
	 * not found.
	 */
	ExceptionCode_NotFound ExceptionCode = 404

	/**
	 * conflict（写并发冲突）.
	 */
	ExceptionCode_Conflict ExceptionCode = 409

	/**
	 * server error（server异常，如超时）.
	 */
	ExceptionCode_ServerError ExceptionCode = 500

	/**
	 * client error（client异常，返回给服务端）.
	 */
	ExceptionCode_ClientError ExceptionCode = -500

	/**
	 * bad gateway（路由异常，如nginx后面的Server挂掉）.
	 */
	ExceptionCode_BadGateway ExceptionCode = 502

	/**
	 * over threshold（超过server端的限流阈值）.
	 */
	ExceptionCode_OverThreshold ExceptionCode = 503

	/**
	 * Server is not started.
	 */
	ExceptionCode_InvalidServerStatus ExceptionCode = 300

	/**
	 * Connection is not registered.
	 */
	ExceptionCode_UnRegister ExceptionCode = 301

	/**
	 * No Handler Found.
	 */
	ExceptionCode_NoHandler ExceptionCode = 302

	ExceptionCode_ResourceNotFound ExceptionCode = -404

	/**
	 * http client error code, ome exceptions that occurred when the use the Nacos RestTemplate and Nacos
	 * AsyncRestTemplate.
	 */
	ExceptionCode_HttpClientErrorCode ExceptionCode = -500
)

type ResponseCode NacosErrorCode

var (
	Response_Success = ResponseCode{
		Code: 200,
		Desc: "Response ok",
	}

	Response_Fail = ResponseCode{
		Code: 500,
		Desc: "Response fail",
	}
)
