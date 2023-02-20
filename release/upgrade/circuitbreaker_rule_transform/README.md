## 熔断规则迁移工具

### 说明

本工具主要用于将存量的熔断规则迁移到1.14.x及以上版本的熔断规则。

### 使用方式

***step1: 编译***

将工具进行编译，再此目录下执行```go build```可完成迁移。

***step2: 升级数据库***

执行数据库升级脚本，将数据库升级到1.14.x及以上版本的数据库，确保数据库中存在circuitbreaker_rule_v2的数据库表。

***step3: 执行迁移***

执行迁移工具，并输入数据库的地址及用户名等信息，如下所示：

```shell
./circuitbreaker_rule_transform --db_addr=127.0.0.1:3306 --db_name=polaris_server --db_user=root --db_pwd=123456
```