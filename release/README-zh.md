# 安装指南

[English](./README.md) | [简体中文](./README-zh.md)

索引：

- [如何安装](#如何安装)
  - [单机版安装](#单机安装)
  - [集群版安装](#集群安装)
- [如何验证](#如何验证)

## 如何安装

单机版和集群版均支持Linux、Mac和Windows各类操作系统，安装包获取路径如下：

- [Github Releases](https://github.com/polarismesh/polaris/releases)
- [Gitee Releases](https://gitee.com/polarismesh/polaris/releases)

### 单机版安装

**Linux操作系统环境**

请下载名称为 `polaris-standalone-release-*.linux.*.zip`的包。

```
unzip polaris-standalone-release-*.linux.*.zip

cd polaris-standalone-release-*.linux.*

bash install.sh
```

**Mac操作系统环境**

请下载名称为 `polaris-standalone-release-*.darwin.*.zip`的包。

```
unzip polaris-standalone-release-*.darwin.*.zip

cd polaris-standalone-release-*.darwin.*

bash install.sh
```

**Windows操作系统环境**

请下载名称为 `polaris-standalone-release-*.windows.*.zip`的包。

```
unzip polaris-standalone-release-*.windows.*.zip

cd polaris-standalone-release-*.windows.*

install.bat
```

### 集群版安装

todo

## 如何验证

可以使用浏览器访问`http://127.0.0.1:8090`，或在命令行下执行下面的命令来验证安装是否成功：

```
curl http://127.0.0.1:8090
```

如果显示"Polaris Server"则表示安装成功。



