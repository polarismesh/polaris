## Docker-Compose 单机版本

### 服务组成

- polaris-server 北极星核心服务
- polaris-console 北极星控制台
- mysql 替换默认的boltdb存储
- polaris-prometheus
- polaris-pushgateway
- grafana

### 创建存储卷

创建mysql、redis 存储卷，方便数据持久化

```shell
docker volume create --name=vlm_data_mysql
```

### 启动服务

```shell
cd polaris/deploy/standalone/docker-compose
docker-compose up -d
```

第一次执行，mysql 启动会导入`polaris-server`的数据库SQL，需要一定启动时间，如果发现服务启动失败，简单起见，通过`restart`实现自动
重启`polaris-server`。

```shell
cd polaris/deploy/standalone/docker-compose
docker-compose up -d polaris-server
```

### 停止服务

```shell
cd polaris/deploy/standalone/docker-compose
docker-compose stop
```

### 清理服务

```shell
cd polaris/deploy/standalone/docker-compose
docker-compose down
```

### 释放存储卷

```shell
docker volume rm vlm_data_mysql
```

### 访问

浏览器访问`http://localhost:8080/#`进入管控台