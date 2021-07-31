# Polaris: Service Discovery and Governance Center

查看 [中文版](https://github.com/PolarisMesh/polaris/blob/master/README-zh.md)

---

Polaris is an operation centre that supports multiple programming languages, with high compatibility to different application framework. 
It supports accessing with SDK or sidecar proxy.

## Overview
Polaris's operation features provided are based on the dimension of the service, 
Polaris's service can actualize industry standard framework and service platform, like [gRPC]，[SPRING CLOUD]，and [Kubernetes Service]. 
These applications can switch in Polaris with no downtime.

Polaris provide features listed as below：

* ** Service Data Management

    Bringing visibility to the control panel, admin can configure HTTP port (label, health status, instance information, policy).

* ** Registration and Discovery

    Provide multi-protocol(HTTP,gRPC), self-registration, and caller server's ability to discover and distribute other server end's data for invocation.

* ** Health Check

    Provide multi-protocol(HTTP,gRPC), provide heartbeat report, server end will monitor heartbeat record, configure overtime health status.
    
## Quick Guide

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
