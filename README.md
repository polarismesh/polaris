# Polaris: Service Discovery and Governance

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/codecov.yaml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/codecov.yaml)
[![codecov.io](https://codecov.io/gh/polarismesh/polaris/branch/main/graph/badge.svg)](https://codecov.io/gh/polarismesh/polaris?branch=main)
[![Contributors](https://img.shields.io/github/contributors/polarismesh/polaris)](https://github.com/polarismesh/polaris/graphs/contributors)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

<img src="logo.svg" width="10%" height="10%" />

English | [简体中文](./README-zh.md)

---

README：

- [Introduction](#introduction)
- [Components](#components)
- [Getting started](#getting-started)
- [Chat group](#chat-group)
- [Contribution](#contribution)

Principle or user manual can visit [website](https://polarismesh.cn/) to learn more

## Introduction

<img src="https://raw.githubusercontent.com/polarismesh/website/main/docs/en/doc/What%20is%20Polaris/Picture/Overview/first-image.png" width="800" />

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

## Getting started

### Installation

#### Download package

You can download the latest standalone package from the addresses below, be aware of to choose the package named ```polaris-standalone-release-*.zip```, and filter the packages by os (windows10: windows, mac: darwin, Linux/Unix: linux).

- [github](https://github.com/polarismesh/polaris/releases)
- [gitee](https://gitee.com/polarismesh/polaris/releases)

Take ```polaris-standalone-release_v1.11.0-beta.2.linux.amd64.zip``` for example, you can use the following commands to unzip package:

```
unzip polaris-standalone-release_v1.11.0-beta.2.linux.amd64.zip
cd polaris-standalone-release_v1.11.0-beta.2.linux 
```

#### Start server

Under Linux/Unix/Mac platform, use those commands to start polaris standalone server:

```
./install.sh
```

Under Windows platform, use those commands to start polaris standalone server:

```
install.bat
```

#### Verify installation

```shell script
curl http://127.0.0.1:8090
```

Return text is 'Polaris Server', proof features run smoothly

If you want to learn more installation methods (changing ports, docker installation, cluster instanllation ect.), please refer: [Installation Guide](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/%E5%AE%89%E8%A3%85%E6%9C%8D%E5%8A%A1%E7%AB%AF/%E5%AE%89%E8%A3%85%E5%8D%95%E6%9C%BA%E7%89%88.html#%E5%8D%95%E6%9C%BA%E7%89%88%E5%AE%89%E8%A3%85)

### Examples

Polaris supports microservices built with multi-language, multi-framework, multi-mode (proxyless / proxy)  to access。

#### Service Discovery and HealthCheck

(1) rpc framework examples:

- [Spring Cloud/Spring Boot Example](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples/polaris-discovery-example)
- [gRPC-Go Example](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/quickstart)

(2) multi-language examples:

- [Java Example](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/quickstart-example)
- [Go Example](https://github.com/polarismesh/polaris-go/tree/main/examples/quickstart)
- [C++ Example](https://github.com/polarismesh/polaris-cpp/tree/main/examples/quickstart)

(3) proxy mode examples:

- [Envoy Example](https://github.com/polarismesh/examples/tree/main/servicemesh/extended-bookinfo)

- DNS Example (Developing)

#### RateLimit

(1) rpc framework examples:

- [Spring Cloud/Spring Boot Example](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples/polaris-ratelimit-example)
- [gRPC-Go Example](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/ratelimit/local)

(2) multi-language examples:

- [Java Example](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/ratelimit-example)
- [Go Example](https://github.com/polarismesh/polaris-go/tree/main/examples/ratelimit)
- [C++ Example](https://github.com/polarismesh/polaris-cpp/tree/main/examples/rate_limit)

(3) proxy mode examples: 

- Envoy Example (Developing)
- Nginx Example (Developing)

#### Flow Control

(1) rpc framework examples:

- [Spring Cloud/Spring Boot Example](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples/polaris-router-featureenv-example)
- [gRPC-Go Example](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/routing/version)

(2) multi-language examples:

- [Java Example](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/router-example/router-multienv-example)
- [Go Example](https://github.com/polarismesh/polaris-go/tree/main/examples/route/dynamic)
- C++ Example (Developing)

(3) proxy mode examples: 

- Envoy Example (Developing)

#### Configuration management

(1) rpc framework examples:

- [Spring Cloud/Spring Boot Example](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples/polaris-config-example)

(2) multi-language examples:

- [Java Example](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/configuration-example)
- [Go Example](https://github.com/polarismesh/polaris-go/tree/main/examples/configuration)

#### More details

More capabilities：[User Manual](https://polarismesh.cn/zh/doc/%E4%BD%BF%E7%94%A8%E6%8C%87%E5%8D%97/%E6%9C%8D%E5%8A%A1%E6%B3%A8%E5%86%8C/%E6%A6%82%E8%BF%B0.html#%E6%A6%82%E8%BF%B0)

## Document

### OpenAPI

[API Manual](https://polarismesh.cn/zh/doc/%E5%8F%82%E8%80%83%E6%96%87%E6%A1%A3/%E6%8E%A5%E5%8F%A3%E6%96%87%E6%A1%A3/%E5%91%BD%E5%90%8D%E7%A9%BA%E9%97%B4%E7%AE%A1%E7%90%86.html#%E5%91%BD%E5%90%8D%E7%A9%BA%E9%97%B4%E7%AE%A1%E7%90%86)

### Benchmark Report

[Benchmark](https://polarismesh.cn/zh/doc/%E5%8F%82%E8%80%83%E6%96%87%E6%A1%A3/%E6%80%A7%E8%83%BD%E6%8A%A5%E5%91%8A/%E6%80%A7%E8%83%BD%E6%B5%8B%E8%AF%95%E6%8A%A5%E5%91%8A.html#polaris%E6%80%A7%E8%83%BD%E6%B5%8B%E8%AF%95%E6%8A%A5%E5%91%8A)

### Official Document

If you want more capablities intruductions, architecture design and best pratises, please refer：[polarismesh.cn](https://polarismesh.cn/zh/doc/%E5%8C%97%E6%9E%81%E6%98%9F%E6%98%AF%E4%BB%80%E4%B9%88/%E7%AE%80%E4%BB%8B.html)

## Chat group

Please scan the QR code to join the chat group.

<img src="https://main.qcloudimg.com/raw/bff4285d70498058caa212805b83a620.jpg" width="30%" height="30%" />

## Contribution

If you have good comments or suggestions, please give us Issues or Pull Requests to contribute to improve the
development experience of Polaris Mesh.

<br>see details：[CONTRIBUTING.md](./CONTRIBUTING.md)
