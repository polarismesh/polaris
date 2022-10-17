# 北极星：服务发现和治理

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/codecov.yaml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/codecov.yaml)
[![codecov.io](https://codecov.io/gh/polarismesh/polaris/branch/main/graph/badge.svg)](https://codecov.io/gh/polarismesh/polaris?branch=main)
[![Docker Pulls](https://img.shields.io/docker/pulls/polarismesh/polaris-server)](https://hub.docker.com/repository/docker/polarismesh/polaris-server/general)
[![Contributors](https://img.shields.io/github/contributors/polarismesh/polaris)](https://github.com/polarismesh/polaris/graphs/contributors)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/polarismesh/polaris?style=flat-square)](https://github.com/polarismesh/polaris)

<img src="logo.svg" width="10%" height="10%" />

[English](./README.md) | 简体中文

---

README：

- [北极星：服务发现和治理](#北极星服务发现和治理)
  - [介绍](#介绍)
  - [快速入门](#快速入门)
    - [安装部署](#安装部署)
      - [下载单机版](#下载单机版)
      - [启动服务端](#启动服务端)
      - [验证安装](#验证安装)
    - [使用样例](#使用样例)
      - [服务注册发现和健康检查](#服务注册发现和健康检查)
      - [服务限流](#服务限流)
      - [流量调度](#流量调度)
      - [配置管理](#配置管理)
      - [更多指南](#更多指南)
  - [详细文档](#详细文档)
    - [OpenAPI参考](#openapi参考)
    - [性能测试报告](#性能测试报告)
    - [官网文档](#官网文档)
  - [交流群](#交流群)
  - [参与贡献](#参与贡献)

北极星原理介绍及相关实践文档请见[北极星官网](https://polarismesh.cn/)

## 介绍

<img src="https://raw.githubusercontent.com/polarismesh/website/main/docs/zh/doc/北极星是什么/图片/简介/第一印象.png" width="800" />

北极星是一个支持多语言、多框架的云原生服务发现和治理中心，解决分布式和微服务架构中的服务可见、故障容错、流量控制和安全问题。

核心功能点：

- <b>服务注册发现和健康检查</b>

  以服务为中心的分布式应用架构中，通过服务和注册发现的方式维护不断变化的请求地址，提高应用的扩展能力，降低应用的迁移成本。北极星提供对注册上来的服务实例进行健康检查，阻止主调方对不健康的服务实例发送请求。
  
- <b>流量调度：路由与负载均衡</b>

  根据请求标签、实例标签和标签匹配规则，对线上流量进行动态调度，可以应用于按地域就近、按标签灰度和新金丝雀发布等多种场景。

- <b>过载保护：服务限流与故障熔断</b>

  对于入口服务，北极星提供服务限流功能，当负载已经超过了系统的最大处理能力时，可针对不同的请求来源和系统资源进行访问限流，避免服务被压垮。
  同时北极星也提供故障熔断功能，根据实时采集的错误率等指标，及时熔断异常的下游服务、接口、实例或者实例分组，降低请求失败率。

- <b>可观测性</b>
  
  提供服务治理可视化监控视图，支持请求量、请求延时和请求成功率的指标查询，支持服务调用关系和多维度的流量曲线查询，实现服务治理功能和流量观测一体化。
  
- <b>配置管理</b>
  
  支持应用配置、公共配置的订阅发布、版本管理、变更通知，实现应用配置动态生效。

特色：

- 北极星的功能采用插件化的形式实现，业务可以根据需求选择使用，也非常容易实现扩展
- 提供SDK和Sidecar两种接入方式，SDK适用于高性能的业务场景，Sidecar适用于无侵入的开发模式
- 对于SDK的接入方式，提供Java、Go、C++等多种语言的客户端，功能实现相同
- 北极星SDK可以集成到常用的框架和网关中，例如Spring Cloud、gRPC和Nginx
- 适用于Kubernetes，支持K8s service和Polaris sidecar的自动注入
- 腾讯百万级服务治理中心的开源版本，沉淀了腾讯多年的分布式服务治理经验

## 快速入门

### 安装部署

#### 下载单机版

可以从以下地址获取最新版本进行下载，下载时候请选择包名为```polaris-standalone-release-*.zip```的软件包，并且根据当前操作系统进行筛选（windows10选择windows，mac选择darwin，Linux/Unix选择linux）。

- [github下载](https://github.com/polarismesh/polaris/releases)
- [gitee下载](https://gitee.com/polarismesh/polaris/releases)

以```polaris-standalone-release_v1.11.0-beta.2.linux.amd64.zip```为例，下载后执行以下命令进行解压：

```
unzip polaris-standalone-release_v1.11.0-beta.2.linux.amd64.zip
cd polaris-standalone-release_v1.11.0-beta.2.linux 
```

#### 启动服务端

在Linux/Unix/Mac平台下，执行以下命令启动北极星单机版：

```
bash install.sh
```

在Windows平台下，执行以下命令启动北极星单机版：

```
install.bat
```

#### 验证安装

```shell
curl http://127.0.0.1:8090
```

返回Polaris Server，证明功能正常

如需了解更多安装方式（如修改安装端口、容器化安装、集群版本安装等），可参考：[部署手册](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/%E5%AE%89%E8%A3%85%E6%9C%8D%E5%8A%A1%E7%AB%AF/%E5%AE%89%E8%A3%85%E5%8D%95%E6%9C%BA%E7%89%88.html#%E5%8D%95%E6%9C%BA%E7%89%88%E5%AE%89%E8%A3%85)

### 使用样例

北极星支持多语言、多框架、多形态（proxyless及proxy）的微服务进行接入。

#### 服务注册发现和健康检查

(1) 基于主流框架开发的微服务接入可参考：

- [Spring Cloud接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/SpringCloud%E5%BA%94%E7%94%A8%E6%8E%A5%E5%85%A5.html#%E6%9C%8D%E5%8A%A1%E6%B3%A8%E5%86%8C)
- [Spring Boot接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/SpringBoot%E5%BA%94%E7%94%A8%E6%8E%A5%E5%85%A5.html#%E6%9C%8D%E5%8A%A1%E6%B3%A8%E5%86%8C)
- [gRPC-Go接入](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/quickstart)

(2) 主流语言微服务接入可参考：

- [Java语言接入](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/quickstart-example)
- [Go语言接入](https://github.com/polarismesh/polaris-go/tree/main/examples/quickstart)
- [C++语言接入](https://github.com/polarismesh/polaris-cpp/tree/main/examples/quickstart)

(3) proxy模式的微服务接入可参考：

- [Envoy接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/Envoy%E7%BD%91%E6%A0%BC%E6%8E%A5%E5%85%A5.html#%E5%BF%AB%E9%80%9F%E6%8E%A5%E5%85%A5)

#### 服务限流

(1) 基于主流框架开发的微服务接入可参考：

- [Spring Cloud接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/SpringCloud%E5%BA%94%E7%94%A8%E6%8E%A5%E5%85%A5.html#%E6%9C%8D%E5%8A%A1%E9%99%90%E6%B5%81)
- [gRPC-Go接入](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/ratelimit/local)

(2) 主流语言微服务接入可参考：

- [Java语言接入](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/ratelimit-example)
- [Go语言接入](https://github.com/polarismesh/polaris-go/tree/main/examples/ratelimit)
- [C++语言接入](https://github.com/polarismesh/polaris-cpp/tree/main/examples/rate_limit)

(3) proxy模式的微服务接入可参考：

- [Nginx接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/Nginx%E7%BD%91%E5%85%B3%E6%8E%A5%E5%85%A5.html#%E8%AE%BF%E9%97%AE%E9%99%90%E6%B5%81)

#### 流量调度

(1) 基于主流框架开发的微服务接入可参考：

- [Spring Cloud接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/SpringCloud%E5%BA%94%E7%94%A8%E6%8E%A5%E5%85%A5.html#%E6%9C%8D%E5%8A%A1%E8%B7%AF%E7%94%B1)
- [gRPC-Go接入](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/routing/version)

(2) 主流语言微服务接入可参考：

- [Java语言接入](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/router-example/router-multienv-example)
- [Go语言接入](https://github.com/polarismesh/polaris-go/tree/main/examples/route/dynamic)

(3) proxy模式的微服务接入可参考：

- [Envoy接入](https://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/Envoy%E7%BD%91%E6%A0%BC%E6%8E%A5%E5%85%A5.html#%E6%B5%81%E9%87%8F%E8%B0%83%E5%BA%A6)

#### 配置管理

(1) 基于主流框架开发的微服务接入可参考：

- [Spring Cloud/Spring Boot接入](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples/polaris-config-example)

(2) 主流语言微服务接入可参考：

- [Java语言接入](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/configuration-example)
- [Go语言接入](https://github.com/polarismesh/polaris-go/tree/main/examples/configuration)

#### 更多指南

更多功能使用指引可参考：[使用指南](https://polarismesh.cn/zh/doc/%E4%BD%BF%E7%94%A8%E6%8C%87%E5%8D%97/%E6%9C%8D%E5%8A%A1%E6%B3%A8%E5%86%8C/%E6%A6%82%E8%BF%B0.html#%E6%A6%82%E8%BF%B0)

## 详细文档

### OpenAPI参考

[接口手册](https://polarismesh.cn/zh/doc/%E5%8F%82%E8%80%83%E6%96%87%E6%A1%A3/%E6%8E%A5%E5%8F%A3%E6%96%87%E6%A1%A3/%E5%91%BD%E5%90%8D%E7%A9%BA%E9%97%B4%E7%AE%A1%E7%90%86.html#%E5%91%BD%E5%90%8D%E7%A9%BA%E9%97%B4%E7%AE%A1%E7%90%86)

### 性能测试报告

[性能报告](https://polarismesh.cn/zh/doc/%E5%8F%82%E8%80%83%E6%96%87%E6%A1%A3/%E6%80%A7%E8%83%BD%E6%8A%A5%E5%91%8A/%E6%80%A7%E8%83%BD%E6%B5%8B%E8%AF%95%E6%8A%A5%E5%91%8A.html#polaris%E6%80%A7%E8%83%BD%E6%B5%8B%E8%AF%95%E6%8A%A5%E5%91%8A)

### 官网文档

如需更多详细功能介绍、架构介绍、最佳实践，可参考：[官网文档入口](https://polarismesh.cn/zh/doc/%E5%8C%97%E6%9E%81%E6%98%9F%E6%98%AF%E4%BB%80%E4%B9%88/%E7%AE%80%E4%BB%8B.html)

## 交流群

扫码下方二维码进入北极星开源社区交流群，加群之前有劳点一下 star，一个小小的 star 是对北极星作者们努力建设社区的动力。

北极星社区相关的特性计划、运营活动都会在交流群中进行发布，加群可以保证您不会错过任何一个关于北极星的资讯。

欢迎您在群里提出你在体验或者使用北极星过程中所遇到的疑问，我们会尽快答复。

您可以在群内提出使用中需要改进的地方，我们评审合理性后会接纳并尽快落实。

如果您发现 bug 请及时提 issue，我们会尽快确认并修改。

<img src="https://main.qcloudimg.com/raw/bff4285d70498058caa212805b83a620.jpg" width="30%" height="30%" />

## 参与贡献

如果你有好的意见或建议，欢迎给我们提 Issues 或 Pull Requests，为提升北极星的开发体验贡献力量。

详见：[CONTRIBUTING.md](CONTRIBUTING.md)
