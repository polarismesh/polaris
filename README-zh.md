# 北极星：服务发现和治理

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/codecov.yaml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/codecov.yaml)
[![codecov.io](https://codecov.io/gh/polarismesh/polaris/branch/main/graph/badge.svg)](https://codecov.io/gh/polarismesh/polaris?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/polarismesh/polaris)](https://goreportcard.com/report/github.com/polarismesh/polaris)
[![Docker Pulls](https://img.shields.io/docker/pulls/polarismesh/polaris-server)](https://hub.docker.com/repository/docker/polarismesh/polaris-server/general)
[![Contributors](https://img.shields.io/github/contributors/polarismesh/polaris)](https://github.com/polarismesh/polaris/graphs/contributors)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/polarismesh/polaris?style=flat-square)](https://github.com/polarismesh/polaris)

<img src="logo.svg" width="10%" height="10%" />

[English](./README.md) | 简体中文

README：

- [北极星：服务发现和治理](#北极星服务发现和治理)
  - [介绍](#介绍)
  - [如何安装](#如何安装)
  - [如何开发服务](#如何开发服务)
  - [如何集成服务网关](#如何集成服务网关)
  - [交流群](#交流群)

更多文档请查看[北极星官网](https://polarismesh.cn)

## 介绍

北极星是一个支持多语言和多框架的服务发现和治理平台，致力于解决分布式和微服务架构中的服务管理、流量管理、故障容错、配置管理和可观测性问题，针对不同的技术栈和环境提供服务治理的标准方案和最佳实践。

<img src="https://raw.githubusercontent.com/polarismesh/website/main/content/zh-cn/docs/北极星是什么/图片/功能特性.png" width="80%" />

**功能**：

- 服务管理：服务注册、服务发现、健康检查
- 流量控制：可自定义的流量路由、负载均衡、限频限流、访问控制
- 故障容错：服务和接口熔断和降级、实例熔断和切换
- 配置管理：版本管理、灰度发布、动态更新

**亮点**：

- 一站式服务治理平台，覆盖注册中心、服务网格和配置中心的能力
- 提供 SDK、开发框架、Java agent 和 sidecar 等多种模式的数据面
- 支持常用的开发框架，例如：Spring Cloud、Dubbo 和 gRPC 等
- 支持 K8s 服务注册和 sidecar 自动注入，实现 Proxy 服务网格

## 如何安装

更多文档请查看[安装指南](https://github.com/polarismesh/polaris/tree/main/release)

## 如何开发服务

北极星提供 SDK、开发框架、Java agent 和 sidecar 等多种模式的数据面。用户可以根据业务需求使用一种或者多种模式的数据面。

使用北极星多语言 SDK，直接调用北极星客户端 API：

- [Polaris Java](https://github.com/polarismesh/polaris-java)
- [Polaris Go](https://github.com/polarismesh/polaris-go)
- [Polaris C/C++](https://github.com/polarismesh/polaris-cpp)
- [Polaris PHP](https://github.com/polarismesh/polaris-php)
- [Polaris Lua](https://github.com/polarismesh/polaris-lua)

使用集成北极星 Java SDK 的 HTTP 和 RPC 框架：

- [spring cloud](https://github.com/Tencent/spring-cloud-tencent)
- [spring boot](https://github.com/polarismesh/spring-boot-polaris)
- dubbo-java
  - [registry and discovery](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-registry-extensions)
  - [routing and load balance](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-cluster-extensions)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-spi-extensions/tree/master/dubbo-filter-extensions)
- [grpc-java](https://github.com/polarismesh/grpc-java-polaris)

使用集成北极星 Go SDK 的 HTTP or RPC 框架：

- dubbo-go
  - [registry and discovery](https://github.com/apache/dubbo-go/tree/main/registry)
  - [routing](https://github.com/apache/dubbo-go/tree/main/cluster/router)
  - [circuit breaker and rate limiter](https://github.com/apache/dubbo-go/tree/main/filter)
  - [examples](https://github.com/apache/dubbo-go-samples/tree/master/polaris)
- [grpc-go](https://github.com/polarismesh/grpc-go-polaris)

使用 K8s 服务注册和 sidecar 自动注入:

- [Polaris Controller](https://github.com/polarismesh/polaris-controller)
- [Polaris Sidecar](https://github.com/polarismesh/polaris-sidecar)

## 如何集成服务网关

用户可以在多种服务网关里集成北极星的服务发现和治理能力。

- [spring cloud gateway](https://github.com/Tencent/spring-cloud-tencent)
- [nginx gateway](https://github.com/polarismesh/nginx-gateway)

## 交流群

扫码二维码，加入北极星开源交流群。欢迎用户反馈使用问题和优化建议。

<img src="./qrcode.png" width="20%" height="20%" />
