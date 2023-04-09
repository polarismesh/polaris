# Polaris: Service Discovery and Governance

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/codecov.yaml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/codecov.yaml)
[![codecov.io](https://codecov.io/gh/polarismesh/polaris/branch/main/graph/badge.svg)](https://codecov.io/gh/polarismesh/polaris?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/polarismesh/polaris)](https://goreportcard.com/report/github.com/polarismesh/polaris)
[![Docker Pulls](https://img.shields.io/docker/pulls/polarismesh/polaris-server)](https://hub.docker.com/repository/docker/polarismesh/polaris-server/general)
[![Contributors](https://img.shields.io/github/contributors/polarismesh/polaris)](https://github.com/polarismesh/polaris/graphs/contributors)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/polarismesh/polaris?style=flat-square)](https://github.com/polarismesh/polaris)

<img src="logo.svg" width="10%" height="10%" />

English | [简体中文](./README-zh.md)

---

README：

- [Introduction](#introduction)
- [How to install](#how-to-install)
- [How to develop service](#how-to-develop-service)
- [How to integrate gateway with Polairs](#how-to-integrate-gateway-with-polairs)
- [Chat group](#chat-group)

visit [website](https://polarismesh.cn/) to learn more

## Introduction

Polaris is a cloud-native service discovery and governance center. It can be used to solve the problem of service
connection, fault tolerance, traffic control and secure in distributed and microservice architecture.

Functions:

- <b>service discover, service register and health check</b>

  Register node addresses into service dynamically, and discover the addresses through the discovery mechnism. Also provide health-checking mechanism to remove the unhealthy instances from service in time. 

- <b>traffic control: request route and load balance</b>

  Provide the mechanism to filter instances by request labels, instances metadata. Users can define rules to direct the request flowing into the locality nearby instances, or gray releasing version instances, etc.

- <b>overload protection: circuit break and rate limit</b>

  Provide the mechanism to reduce the request rate when burst request flowing into the entry services.

  Provide the mechanism to collect the healthy statistic by the response, also kick of the services/interfaces/groups/instances when they are unhealthy.

- <b>observability</b>

  User can see the metrics and tracing through the vison diagram, to be aware of the api call status on time.

- <b>config management</b>

  Provide the mechanism to dynamic configuration subscribe, version management, notify change, to apply the configuration to application in time.

Features:

- It provides SDK for high-performance business scenario and sidecar for non-invasive development mode.
- It provides multiple clients for different development languages, such as Java, Go, C++ and Nodejs.
- It can integrate with different service frameworks and gateways, such as Spring Cloud, gRPC and Nginx.
- It is compatible with Kubernetes and supports automatic injection of K8s service and Polaris sidecar.

## How to install 

Here is how to install the standalone version. Visit [Installation Guide](https://polarismesh.cn/docs/使用指南/服务端安装) to learn more.

The release packages of standalone and cluster have been provided for Linux, Mac and Windows.

- [Github Releases](https://github.com/polarismesh/polaris/releases)
- [Gitee Releases](https://gitee.com/polarismesh/polaris/releases)

Download the release package of last standalone version from Github or Gitee Releases.

**Linux**

Download the package named `polaris-standalone-release-*.linux.*.zip`.

```
unzip polaris-standalone-release-*.linux.*.zip

cd polaris-standalone-release-*.linux.*

bash install.sh
```

**Mac**

Download the package named `polaris-standalone-release-*.darwin.*.zip`.

```
unzip polaris-standalone-release-*.darwin.*.zip

cd polaris-standalone-release-*.darwin.*

bash install.sh
```

**Windows**

Download the package named `polaris-standalone-release-*.windows.*.zip`.

```
unzip polaris-standalone-release-*.windows.*.zip

cd polaris-standalone-release-*.windows.*

install.bat
```

Run the command to verify the installation.

```
curl http://127.0.0.1:8090
```

If the response is "Polaris Server", the installation is successful.

## How to develop service

Polaris provides multi-mode data plane including SDK, development framework, Java agent and sidecar. You can select one or more mode to develop service according to business requirements. 

The first three modes are used in proxyless service governance solution that has lower performance loss and resource cost. The last one is used in proxy service mesh solution that has lower coupling level.

Use Polaris multi-language SDK and call Polaris Client API directly:

- [Polaris Java](https://github.com/polarismesh/polaris-java)
- [Polaris Go](https://github.com/polarismesh/polaris-go)
- [Polaris C/C++](https://github.com/polarismesh/polaris-cpp)
- [Polaris PHP](https://github.com/polarismesh/polaris-php)
- [Polaris Lua](https://github.com/polarismesh/polaris-lua)

Use HTTP or RPC frameworks already integrating Polaris Java SDK:

- [spring cloud](https://github.com/Tencent/spring-cloud-tencent)
- [spring boot](https://github.com/polarismesh/spring-boot-polaris)
- dubbo-java
  - [registry, discovery and routing](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-registry-extensions)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-filter-extensions)
- [grpc-java](https://github.com/polarismesh/grpc-java-polaris)

Use HTTP or RPC frameworks already integrating Polaris Go SDK:

- dubbo-go
  - [registry, discovery and routing](https://github.com/apache/dubbo-go/tree/main/registry)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-go/tree/main/filter)
  - [examples](https://github.com/apache/dubbo-go-samples/tree/master/polaris)
- [grpc-go](https://github.com/polarismesh/grpc-go-polaris)

Use K8s service and sidecar:

- [Polaris Controller](https://github.com/polarismesh/polaris-controller)
- [Polaris Sidecar](https://github.com/polarismesh/polaris-sidecar)

## How to integrate gateway with Polairs

Gateway is important in distributed or microservice architecture. You can integrate multiple gateway with Polaris.

- [spring cloud gateway](https://github.com/Tencent/spring-cloud-tencent)
- [nginx gateway](https://github.com/polarismesh/nginx-gateway)

## Chat group

Please scan the QR code to join the chat group.

<img src="https://main.qcloudimg.com/raw/bff4285d70498058caa212805b83a620.jpg" width="20%" height="20%" />
