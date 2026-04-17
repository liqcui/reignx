# ReignX 测试套件

本目录包含 ReignX 项目的所有测试代码。

## 目录结构

```
tests/
├── unit/           # 单元测试 - 测试单个函数/方法
├── integration/    # 集成测试 - 测试多个组件协同工作
└── e2e/            # 端到端测试 - 测试完整的用户场景
```

## 测试类型说明

### 单元测试 (unit/)
- 测试单个函数、方法或类的行为
- 快速执行，无外部依赖
- 运行命令：`go test ./tests/unit/...`

### 集成测试 (integration/)
- 测试多个组件之间的交互
- 可能需要数据库、gRPC 等外部服务
- 当前包含：
  - `integration_test.go` - 完整的集成测试套件
  - `test_encryption_flow.go` - 加密流程测试
  - `test_decryption_demo.go` - 解密演示测试
- 运行命令：`go test ./tests/integration/...`

### 端到端测试 (e2e/)
- 测试完整的用户场景和工作流
- 需要完整的系统环境（数据库、API、Web 服务器等）
- 运行命令：`go test ./tests/e2e/...`

## 运行所有测试

```bash
# 运行所有测试
go test ./tests/...

# 运行测试并显示详细输出
go test -v ./tests/...

# 运行测试并生成覆盖率报告
go test -cover ./tests/...
go test -coverprofile=coverage.out ./tests/...
go tool cover -html=coverage.out
```

## 测试最佳实践

1. 遵循 Go 测试命名规范：`TestXxx` 或 `Test_xxx`
2. 使用表驱动测试（table-driven tests）处理多个测试用例
3. 集成测试使用 `_test` 包名以避免循环依赖
4. 使用 `testing.T.Helper()` 标记辅助函数
5. 清理测试资源，使用 `t.Cleanup()` 或 `defer`
