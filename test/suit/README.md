# Polaris UnitTest Suit 接入

## 第三方存储插件接入单元测试体系

- 实现 `TestDataClean` 接口
  - 在 `test/suit` 目录下写入一个 `go` 文件
  - 利用 `go _ import` 执行 `init` 方法的机制，调用 `testsuit.SetTestDataClean` 注入构建 `TestDataClean` 的 supplier 行数
- 测试依赖第三方存储插件时的北极星启动配置文件 `polaris-server.yaml` 的路径信息，通过设置环境变量 `POLARIS_TEST_BOOTSTRAP_FILE`

具体参考 [polaris-contrib/polaris-store-postgresql](https://github.com/polaris-contrib/polaris-store-postgresql)
