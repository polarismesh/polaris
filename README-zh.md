# 北极星：服务发现和治理

[![Build Status](https://github.com/polarismesh/polaris/actions/workflows/testing.yml/badge.svg)](https://github.com/PolarisMesh/polaris/actions/workflows/testing.yml)
[![codecov.io](https://codecov.io/gh/polarismesh/polaris/branch/main/graph/badge.svg)](https://codecov.io/gh/polarismesh/polaris?branch=main)
[![Contributors](https://img.shields.io/github/contributors/polarismesh/polaris)](https://github.com/polarismesh/polaris/graphs/contributors)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

<img src="logo.svg" width="10%" height="10%" />

[English](./README.md) | 简体中文

---

README：

- [介绍](#介绍)
- [项目构成](#项目构成)
- [快速入门](#快速入门)
- [交流群](#交流群)
- [参与贡献](#参与贡献)

北极星原理介绍及相关实践文档请见[北极星官网](https://polarismesh.cn/zh/doc/)

## 介绍

北极星是一个支持多语言、多框架的云原生服务发现和治理中心，解决分布式和微服务架构中的服务可见、故障容错、流量控制和安全问题。

功能：

- 基础功能：服务发现、服务注册、健康检查
- 故障容错：熔断降级、访问限流
- 流量控制：动态路由、负载均衡
- 安全：访问鉴权

特色：

- 北极星的功能采用插件化的形式实现，业务可以根据需求选择使用，也非常容易实现扩展
- 提供SDK和Sidecar两种接入方式，SDK适用于高性能的业务场景，Sidecar适用于无侵入的开发模式
- 对于SDK的接入方式，提供Java、Go、C++和NodeJS等多种语言的客户端，功能实现相同
- 北极星SDK可以集成到常用的框架和网关中，例如Spring Cloud、gRPC和Nginx
- 适用于Kubernetes，支持K8s service和Polaris sidecar的自动注入
- 腾讯百万级服务治理中心的开源版本，沉淀了腾讯多年的分布式服务治理经验

## 项目构成

服务端:

- [polaris](https://github.com/PolarisMesh/polaris): 控制面
- [polaris-console](https://github.com/PolarisMesh/polaris-console): 控制台

客户端:

- [polaris-java](https://github.com/PolarisMesh/polaris-java): Java客户端
- [polaris-go](https://github.com/PolarisMesh/polaris-go): Go客户端
- [polaris-cpp](https://github.com/PolarisMesh/polaris-cpp): C++客户端
- [polaris-php](https://github.com/polarismesh/polaris-php): PHP客户端
- [polaris-sidecar](https://github.com/PolarisMesh/polaris-sidecar): 基于Envoy的Sidecar

生态组件:

- [polaris-controller](https://github.com/PolarisMesh/polaris-controller): K8s控制器，支持K8s Service和Polaris Sidecar自动注入
- [spring-cloud-tencent](https://github.com/Tencent/spring-cloud-tencent): spring cloud集成polaris-java
- [grpc-java-polaris](https://github.com/PolarisMesh/grpc-java-polaris): grpc-java集成polaris-java
- [grpc-go-polaris](https://github.com/PolarisMesh/grpc-go-polaris): grpc-go集成polaris-go
- [dubbo3/dubbo-go](https://github.com/polarismesh/examples/tree/main/dubbo3/dubbogo): dubbo-go集成polaris-go
- [nginx-polaris](https://github.com/PolarisMesh/nginx-polaris): nginx集成polaris-cpp

其他:

- [website](https://github.com/PolarisMesh/website): 官网
- [samples](https://github.com/PolarisMesh/samples): 示例

## 快速入门

### 前置准备

#### 准备数据库

需要下载并安装MySQL，版本号要求>=5.7，可以在这里进行下载：https://dev.mysql.com/downloads/mysql/5.7.html

#### 导入数据库建表脚本

建表脚本为./store/sqldb/scripts/polaris_server.sql，可通过mysql命令或者admin客户端进行导入

#### 准备golang编译环境

北极星服务端编译需要golang编译环境，版本号要求>=1.17，可以在这里进行下载：https://golang.org/dl/#featured

### 编译构建

```shell
chmod +x build.sh
./build.sh
```

构建完后，可以在当前目录看到polaris-server-release_${version}.tar.gz的软件包。

### 安装

#### 解压软件包

获取polaris-server-release_${version}.tar.gz，并解压

#### 修改数据库配置

进入解压后的目录，打开polaris-server.yaml，替换DB配置相关的几个变量为实际的数据库参数；##DB_USER##（数据库用户名），##DB_PWD##（数据库密码），##DB_ADDR##（数据库地址），##DB_NAME##（数据库名称）

#### 执行安装脚本

```shell
chmod +x ./tool/*.sh
#进行安装
./tool/start.sh
#测试进程是否启动成功
./tool/p.sh
```

最后一步运行p.sh后，返回Polaris Server，证明启动成功。

#### 验证安装

```shell
curl http://127.0.0.1:8090
```

返回Polaris Server，证明功能正常

### 接入

北极星支持多语言、多框架、多形态（proxyless及proxy）的微服务进行接入。

(1) 主流语言微服务接入可参考：

- [Java语言快速接入样例](https://github.com/polarismesh/polaris-java/tree/main/polaris-examples/quickstart-example)
- [Go语言快速接入样例](https://github.com/polarismesh/polaris-go/tree/main/examples/quickstart)
- [C++语言快速接入样例](https://github.com/polarismesh/polaris-cpp/tree/main/examples/quickstart)

(2) 基于主流框架开发的微服务接入可参考：

- [Spring Cloud快速接入样例](https://github.com/Tencent/spring-cloud-tencent/tree/main/spring-cloud-tencent-examples)
- [Spring Boot快速接入样例](https://github.com/polarismesh/spring-boot-polaris/tree/main/spring-boot-polaris-examples/quickstart-example)
- [gRPC-Go快速接入样例](https://github.com/polarismesh/grpc-go-polaris/tree/main/examples/quickstart)
- [gRPC-Java快速接入样例](https://github.com/polarismesh/grpc-java-polaris/tree/main/grpc-java-polaris-examples/quickstart-example)

(3) proxy模式的微服务接入可参考：

- [Envoy快速接入样例](https://github.com/polarismesh/examples/tree/main/servicemesh/extended-bookinfo)

更多接入指引可参考：[接入文档](https://polarismesh.cn/zh/doc/快速入门/使用SDK接入.html#使用-sdk%20接入)

## 使用指南

北极星各部分功能的使用方式可参考：[使用指南](https://polarismesh.cn/zh/doc/使用指南/基本原理.html#基本原理)

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