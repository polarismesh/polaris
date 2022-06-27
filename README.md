# Polaris: Service Discovery and Governance

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/testing.yml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/testing.yml)
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

Principle or user manual can visit [website](https://polarismesh.cn/zh/doc/) to learn more

## Introduction

Polaris is a cloud-native service discovery and governance center. It can be used to solve the problem of service
connection, fault tolerance, traffic control and secure in distributed and microservice architecture.

Functions:

- basic: service discover, service register and health check
- fault tolerance: circuit break and rate limit
- traffic control: request route and load balance
- secure: authenticate

Features:

- It provides SDK for high-performance business scenario and sidecar for non-invasive development mode.
- It provides multiple clients for different development languages, such as Java, Go, C++ and Nodejs.
- It can integrate with different service frameworks and gateways, such as Spring Cloud, gRPC and Nginx.
- It is compatible with Kubernetes and supports automatic injection of K8s service and Polaris sidecar.

## Components

server:

- [polaris](https://github.com/PolarisMesh/polaris): Control Plane
- [polaris-console](https://github.com/PolarisMesh/polaris-console): Console

client:

- [polaris-java](https://github.com/PolarisMesh/polaris-java): Java Client
- [polaris-go](https://github.com/PolarisMesh/polaris-go): Go Client
- [polaris-cpp](https://github.com/PolarisMesh/polaris-cpp): C++ Client
- [polaris-php](https://github.com/polarismesh/polaris-php): PHP Client
- [polaris-sidecar](https://github.com/PolarisMesh/polaris-sidecar): Envoy based Sidecar

ecosystem:

- [polaris-controller](https://github.com/PolarisMesh/polaris-controller): K8s Controller for Automatic Injection of K8s
  Service and Polaris Sidecar
- [spring-cloud-tencent](https://github.com/Tencent/spring-cloud-tencent): spring cloud integrates with polaris-java
- [grpc-java-polaris](https://github.com/PolarisMesh/grpc-java-polaris): grpc-java integrates with polaris-java
- [grpc-go-polaris](https://github.com/PolarisMesh/grpc-go-polaris): grpc-go integrates with polaris-go
- [dubbo3/dubbo-go](https://github.com/polarismesh/examples/tree/main/dubbo3/dubbogo): dubbo-go integrates with
  polaris-go
- [nginx-polaris](https://github.com/PolarisMesh/nginx-polaris): nginx integrates with polaris-cpp

others:

- [website](https://github.com/PolarisMesh/website): Source for the polarismesh.cn site
- [samples](https://github.com/PolarisMesh/samples): Samples for Learning PolarisMesh

## Getting started

### Preconditions

#### Prepare database

Please download and install MySQL, version requirement >=5.7, download available here:
https://dev.mysql.com/downloads/mysql/5.7.html

#### Import SQL script

Point Script: ./store/sqldb/scripts/polaris_server.sql, one can import through mysql admin or console.

#### Prepare golang compile environment

Polaris server end needs golang compile environment, version number needs >=1.17, download available
here: https://golang.org/dl/#featured.

### Build

```shell script
chmod +x build.sh
./build.sh
```

After built, one can see 'polaris-server-release_${version}.tar.gz' package from the list.

### Installation

#### Unzip package

Obtain polaris-server-release_${version}.tar.gz, and unzip.

#### Change polaris configuration

After unzipped, vi polaris-server.yaml, replace DB configuration's variable to real database information
: ##DB_USER## (database username), ##DB_PWD##（database password）, ##DB_ADDR##（database address）, ##DB_NAME##（database
name）

#### Execute Installation Script

```shell script
chmod +x ./tool/*.sh
# install
./tool/start.sh
# test whether the process is successful 
./tool/p.sh
```

After all, run ./p.sh, prompt Polaris Server, proof the installation is successful

#### Verify installation

```shell script
curl http://127.0.0.1:8090
```

Return text is 'Polaris Server', proof features run smoothly

## How to access

Polaris supports microservices built with multi-language, multi-framework, multi-mode (proxyless / proxy)  to access。

(1) multi-language access guide：

- [Java QuickStart Example](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/quickstart-example)
- [Go QuickStart Example](https://github.com/polarismesh/polaris-go/tree/main/examples/quickstart)
- [C++ QuickStart Example](https://github.com/polarismesh/polaris-cpp/tree/main/examples/quickstart)

(2) multi-framework access guide:

- [Spring Cloud Examples](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples)
- [Spring Boot QuickStart Example](https://github.com/polarismesh/spring-boot-polaris/tree/main/spring-boot-polaris-examples/quickstart-example)
- [gRPC-Go QuickStart Example](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/quickstart)
- [gRPC-Java QuickStart Example](https://github.com/polarismesh/grpc-java-polaris/tree/main/grpc-java-polaris-examples/quickstart-example)

(3) proxy mode access guide：

- [Envoy QuickStart Example](https://github.com/polarismesh/examples/tree/main/servicemesh/extended-bookinfo)

More access guide：[Doc](https://polarismesh.cn/zh/doc/快速入门/使用SDK接入.html#使用-sdk%20接入)

## User manual

Polaris function guide can refer：[Manual](https://polarismesh.cn/zh/doc/使用指南/基本原理.html#基本原理)

## Chat group

Please scan the QR code to join the chat group.

<img src="https://main.qcloudimg.com/raw/bff4285d70498058caa212805b83a620.jpg" width="30%" height="30%" />

## Contribution

If you have good comments or suggestions, please give us Issues or Pull Requests to contribute to improve the
development experience of Polaris Mesh.

<br>see details：[CONTRIBUTING.md](./CONTRIBUTING.md)