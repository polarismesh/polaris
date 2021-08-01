# 北极星：服务发现和治理

<img src="logo.png" width="10%" height="10%" />

---

README：

- [介绍](#介绍)
- [快速入门](#快速入门)

其他文档请见[北极星官网](https://polarismesh.cn)

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

## 快速入门

### 前置准备

#### 准备数据库

需要下载并安装MySQL，版本号要求>=5.7，可以在这里进行下载：https://dev.mysql.com/downloads/mysql/5.7.html

#### 导入数据库建表脚本

建表脚本为./store/defaultStore/polaris_server.sql，可通过mysql命令或者admin客户端进行导入

#### 准备golang编译环境

北极星服务端编译需要golang编译环境，版本号要求>=1.12，可以在这里进行下载：https://golang.org/dl/#featured

### 编译构建

````shell script
chmod +x build.sh
./build.sh
````
构建完后，可以在当前目录看到polaris-server-release_${version}.tar.gz的软件包。

### 安装

#### 解压软件包

获取polaris-server-release_${version}.tar.gz，并解压

#### 修改数据库配置

进入解压后的目录，打开polaris-server.yaml，替换DB配置相关的几个变量为实际的数据库参数；##DB_USER##（数据库用户名），##DB_PWD##（数据库密码），##DB_ADDR##（数据库地址），##DB_NAME##（数据库名称）

#### 执行安装脚本

````shell script
chmod +x ./tool/*.sh
#进行安装
./tool/install.sh
#测试进程是否启动成功
./tool/p.sh
````
最后一步运行p.sh后，返回Polaris Server，证明启动成功。

#### 验证安装

````shell script
curl http://127.0.0.1:8080
```` 
返回Polaris Server，证明功能正常

## License

The polaris is licensed under the BSD 3-Clause License. Copyright and license information can be found in the file [LICENSE](LICENSE)
