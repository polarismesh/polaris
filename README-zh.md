# 北极星：服务发现和治理

<img src="logo.png" width="10%" height="10%" />

---

README包含：

- [简介](#简介)
- [快速入门](#快速入门)

其他文档请见[北极星官网](https://polarismesh.cn)

## 介绍

北极星是一个支持多语言、多框架的云原生服务发现和治理中心，支持高性能SDK和无侵入Sidecar两种使用方式。

北极星的治理功能是基于服务维度来提供的，北极星的服务可对应到业界主流的框架/平台服务的实现，如[gRPC]，[SPRING CLOUD]，以及[Kubernetes Service]。基于这些框架/平台开发的应用可以快速接入北极星服务治理。

北极星服务端提供以下主流功能特性：

* ** 服务数据管理

    执行可视化控制台或者管理员基于HTTP管理端接口对于服务数据（标签，健康状态，实例信息，治理规则）进行读写操作。

* ** 服务注册发现

    提供多协议（HTTP,gRPC）接口供被调端服务进行自注册，以及主调端应用发现并拉取其他被调端服务的服务数据，以便接下来进行服务调用。

* ** 健康检查

    提供多协议（HTTP,gRPC）接口供被调端进行心跳上报，服务端会实时监测心跳记录，对超时的实例进行健康状态变更。
    
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
