# Polaris: Service Discovery and Governance

<img src="logo.png" width="10%" height="10%" />

* [简体中文](https://github.com/PolarisMesh/polaris/blob/master/README-zh.md)

---

README：

- [Introduction](#introduction)
- [Quick Start](#quick-start)

Visit [website](https://polarismesh.cn) to learn more

## Introduction

Polaris is a cloud-native service discovery and governance center. It can be used to solve the problem of service connection, fault tolerance, traffic control and secure in distributed and microservice architecture.

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

## Quick Start

### Preconditions

#### Prepare database 

Please download and install MySQL, version requirement >=5.7, download available here: 
https://dev.mysql.com/downloads/mysql/5.7.html

#### Import SQL script

Point Script: ./store/defaultStore/polaris_server.sql, one can import through mysql admin or console.

#### Prepare golang compile environment

Polaris server end needs golang compile environment, version number needs >=1.12, download available here: https://golang.org/dl/#featured.

### Build

````shell script
chmod +x build.sh
./build.sh
````
After built, one can see 'polaris-server-release_${version}.tar.gz' package from the list. 

### Installation

#### Unzip package

Obtain polaris-server-release_${version}.tar.gz, and unzip.

#### Change polaris configuration

After unzipped, vi polaris-server.yaml, replace DB configuration's variable to real database information
: ##DB_USER## (database username), ##DB_PWD##（database password）, ##DB_ADDR##（database address）, ##DB_NAME##（database name）

#### Execute Installation Script

````shell script
chmod +x ./tool/*.sh
# install
./tool/install.sh
# test whether the process is successful 
./tool/p.sh
````
After all, run ./p.sh, prompt Polaris Server, proof the installation is successful 

#### Verify installation

````shell script
curl http://127.0.0.1:8080
```` 
Return text is 'Polaris Server', proof features run smoothly 

## License

The polaris is licensed under the BSD 3-Clause License. Copyright and license information can be found in the file [LICENSE](LICENSE)
