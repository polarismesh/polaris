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

README：

- [Polaris: Service Discovery and Governance](#polaris-service-discovery-and-governance)
  - [Introduction](#introduction)
  - [How to install](#how-to-install)
  - [How to develop service](#how-to-develop-service)
  - [How to integrate service gateway](#how-to-integrate-service-gateway)
  - [Chat group](#chat-group)

Visit [Website](https://polarismesh.cn/) to learn more

## Introduction

Polaris is an open source system for service discovery and governance. It can be used to solve the problem of service management, traffic control, fault tolerance and config management in distributed and microservice architecture.

<img src="https://raw.githubusercontent.com/polarismesh/website/main/content/en/docs/What%20is%20Polaris/Picture/function.png" width="80%" />

**Functions**:

- service management: service discovery, service registry and health check 
- traffic control: customizable routing, load balance, rate limiting and access control
- fault tolerance: circuit breaker for service, interface and instance
- config management: config version control, grayscale release and dynamic update

**Features**:

- It is a one-stop solution instead of registry center, service mesh and config center.
- It provides multi-mode data plane, including SDK, development framework, Java agent and sidecar.
- It is integrated into the most frequently used frameworks, such as Spring Cloud, Dubbo and gRPC.
- It supports K8s service registry and automatic injection of sidecar for proxy service mesh.

## How to install 

Visit [Installation Guide](https://github.com/polarismesh/polaris/tree/main/release) to learn more

## How to develop service

Polaris provides multi-mode data plane including SDK, development framework, Java agent and sidecar. You can select one or more mode to develop service according to business requirements. 

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
  - [registry and discovery](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-registry-extensions)
  - [routing and load balance](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-cluster-extensions)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-filter-extensions)
- [grpc-java](https://github.com/polarismesh/grpc-java-polaris)

Use HTTP or RPC frameworks already integrating Polaris Go SDK:

- dubbo-go
  - [registry and discovery](https://github.com/apache/dubbo-go/tree/main/registry)
  - [routing](https://github.com/apache/dubbo-go/tree/main/cluster/router)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-go/tree/main/filter)
  - [examples](https://github.com/apache/dubbo-go-samples/tree/master/polaris)
- [grpc-go](https://github.com/polarismesh/grpc-go-polaris)

Use K8s service and sidecar:

- [Polaris Controller](https://github.com/polarismesh/polaris-controller)
- [Polaris Sidecar](https://github.com/polarismesh/polaris-sidecar)

## How to integrate service gateway

You can integrate service gateways with Polaris service discovery and governance.

- [spring cloud gateway](https://github.com/Tencent/spring-cloud-tencent)
- [nginx gateway](https://github.com/polarismesh/nginx-gateway)

## Chat group

Please scan the QR code to join the chat group.

<img src="./qrcode.png" width="20%" height="20%" />
