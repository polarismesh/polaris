# PolarisMesh服务发现性能测试报告

## 测试目的

了解PolarisMesh的服务发现性能负载和容量，帮助用户更快的运用评估PolarisMesh系统负荷，协助用户进行资源选型。

## 测试工具

我们使用开源的ghz工具进行压测。测试工具地址：https://github.com/bojand/ghz/releases/tag/v0.105.0

## 测试场景

- 验证实例注册的性能
- 验证实例心跳上报的性能
- 验证实例查询的性能
- 验证实例反注册的性能

## 测试环境

### 规格1

| 组件    | 参数                              |
| ------- | --------------------------------- |
| polaris | CPU 4核，内存32G，3节点           |
| 数据库  | CPU 4核，内存8G，存储100G，双节点 |
| redis   | xxx                               |

### 规格2

| 组件    | 参数                               |
| ------- | ---------------------------------- |
| polaris | CPU 4核，内存32G，6节点            |
| 数据库  | CPU 8核，内存16G，存储100G，双节点 |
| redis   | xxx                                |

## 测试数据

### 实例注册

- 测试接口：

```go
package v1;

service PolarisGRPC {
 // 被调方注册服务实例
 rpc RegisterInstance(Instance) returns(Response) {}
}
```

- 测试命令：ghz xxxx

#### 规格1

| 机器*并发数 | 实例数 | TPS | RT(ms) | 最小RT(ms) | 最大RT(ms) | polaris负载 | 数据库负载 | redis负载 |
| ----------- | ------ | --- | ------ | ---------- | ---------- | ----------- | ---------- | --------- |
| 1 * 100     |        |     |        |            |            |             |            |           |
| 2 * 100     |        |     |        |            |            |             |            |           |
| 4 * 100     |        |     |        |            |            |             |            |           |
| 8 * 100     |        |     |        |            |            |             |            |           |

#### 规格2

| 机器*并发数 | 实例数 | TPS | RT(ms) | 最小RT(ms) | 最大RT(ms) | polaris负载 | 数据库负载 | redis负载 |
| ----------- | ------ | --- | ------ | ---------- | ---------- | ----------- | ---------- | --------- |
| 1 * 100     |        |     |        |            |            |             |            |           |
| 2 * 100     |        |     |        |            |            |             |            |           |
| 4 * 100     |        |     |        |            |            |             |            |           |
| 8 * 100     |        |     |        |            |            |             |            |           |

### 实例心跳上报

- 测试接口：

```go
package v1;

service PolarisGRPC {
  // 被调方上报心跳
  rpc Heartbeat(Instance) returns(Response) {}
}
```

- 测试命令：

### 实例查询

- 测试接口：

```go
package v1;

service PolarisGRPC {
  // 统一发现接口
  rpc Discover(stream DiscoverRequest) returns(stream DiscoverResponse) {}
}
```

- 测试命令：

### 实例反注册

- 测试接口：

```go
package v1;

service PolarisGRPC {
  // 被调方反注册服务实例
  rpc DeregisterInstance(Instance) returns(Response) {}
}
```

- 测试命令:
