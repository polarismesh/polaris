# Installation Guide

[English](./README.md) | [简体中文](./README-zh.md)

README：

- [How to install](#how-to-install)
  - [Standalone](#standalone)
  - [Cluster](#cluster)
- [How to verify](#how-to-verify)

## How to install

The release packages of standalone and cluster have been provided for Linux, Mac and Windows.

- [Github Releases](https://github.com/polarismesh/polaris/releases)
- [Gitee Releases](https://gitee.com/polarismesh/polaris/releases)

### Standalone

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

### Cluster

todo

## How to verify

Run the command to verify the installation.

```
curl http://127.0.0.1:8090
```

If the response is "Polaris Server", the installation is successful.
