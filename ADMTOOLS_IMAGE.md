# ReignX Admtools Image

基于 `quay.io/openshift-psap-qe/admtools:latest` 的多架构容器镜像，支持 SSH 登录和 ReignX Agent 双模式。

## 特性

- ✅ 基于 OpenShift admtools（包含所有管理工具）
- ✅ 支持 SSH 登录（root 和 reignx 用户）
- ✅ 支持 ReignX Agent 模式
- ✅ 多架构支持（linux/arm64, linux/amd64）
- ✅ 适配 macOS（Apple Silicon M1/M2/M3 和 Intel）
- ✅ 可配置环境变量
- ✅ 开箱即用

## 快速开始

### 1. 构建镜像

```bash
# 构建多架构镜像（使用默认密码：changeme）
./build-admtools.sh

# 构建时设置自定义密码（推荐）
ROOT_PASSWORD="your_root_pass" USER_PASSWORD="your_user_pass" ./build-admtools.sh

# 或指定镜像名称
IMAGE_NAME=my-admtools IMAGE_TAG=v1.0 ./build-admtools.sh
```

**安全提示：** 默认密码为 `changeme`，生产环境请务必通过 `ROOT_PASSWORD` 和 `USER_PASSWORD` 环境变量设置强密码。

### 2. 运行容器

#### 仅 SSH 模式（无 Agent）

```bash
# 使用 Podman
podman run -d --name admtools \
  -p 2222:22 \
  reignx-admtools:latest

# 使用 Docker
docker run -d --name admtools \
  -p 2222:22 \
  reignx-admtools:latest
```

#### SSH + ReignX Agent 模式

```bash
# macOS 上连接到主机上的 ReignX Server
podman run -d --name admtools \
  -p 2222:22 \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=host.docker.internal:50051 \
  -e NODE_ID=admtools-$(date +%s) \
  reignx-admtools:latest

# 或使用实际的服务器地址
podman run -d --name admtools \
  -p 2222:22 \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=192.168.1.100:50051 \
  -e NODE_ID=admtools-macos-001 \
  -e TLS_ENABLED=false \
  reignx-admtools:latest
```

### 3. SSH 登录

```bash
# 使用 root 用户
ssh root@localhost -p 2222
# 密码: changeme (或通过 ROOT_PASSWORD/USER_PASSWORD 设置)

# 使用 reignx 用户
ssh reignx@localhost -p 2222
# 密码: changeme (或通过 ROOT_PASSWORD/USER_PASSWORD 设置)

# 使用 SSH 密钥（推荐）
# 1. 将公钥复制到容器
podman exec admtools mkdir -p /root/.ssh
cat ~/.ssh/id_rsa.pub | podman exec -i admtools tee -a /root/.ssh/authorized_keys

# 2. 无密码登录
ssh root@localhost -p 2222
```

## 环境变量

### ReignX Agent 配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ENABLE_AGENT` | `false` | 是否启用 ReignX Agent |
| `REIGNX_SERVER` | `localhost:50051` | ReignX API 服务器地址 |
| `NODE_ID` | (hostname) | 节点唯一标识符 |
| `TLS_ENABLED` | `false` | 是否启用 TLS |
| `TLS_SKIP_VERIFY` | `false` | 跳过 TLS 证书验证（仅开发） |
| `LOG_LEVEL` | `info` | 日志级别 (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | 日志格式 (json/text) |
| `HEARTBEAT_INTERVAL` | `30s` | 心跳间隔 |
| `METRICS_INTERVAL` | `60s` | 指标收集间隔 |

### TLS 证书配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `TLS_CA_CERT` | `/etc/reignx/ca.crt` | CA 证书路径 |
| `TLS_CLIENT_CERT` | `/etc/reignx/client.crt` | 客户端证书路径 |
| `TLS_CLIENT_KEY` | `/etc/reignx/client.key` | 客户端私钥路径 |

## 高级用法

### 1. 挂载 TLS 证书

```bash
# 准备证书目录
mkdir -p certs
cp ca.crt client.crt client.key certs/

# 运行容器并挂载证书
podman run -d --name admtools \
  -p 2222:22 \
  -v $(pwd)/certs:/etc/reignx:ro \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=api.example.com:50051 \
  -e NODE_ID=admtools-secure-001 \
  -e TLS_ENABLED=true \
  reignx-admtools:latest
```

### 2. 自定义 SSH 密钥

```bash
# 生成 SSH 密钥对
ssh-keygen -t ed25519 -f admtools_key -N ""

# 运行容器并注入公钥
podman run -d --name admtools \
  -p 2222:22 \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=host.docker.internal:50051 \
  reignx-admtools:latest

# 复制公钥到容器
cat admtools_key.pub | podman exec -i admtools tee /root/.ssh/authorized_keys
podman exec admtools chmod 600 /root/.ssh/authorized_keys

# 使用私钥登录
ssh -i admtools_key root@localhost -p 2222
```

### 3. 批量启动多个节点

```bash
#!/bin/bash
# 启动 10 个 admtools 容器作为测试节点

for i in {1..10}; do
    PORT=$((2222 + i))
    podman run -d --name admtools-$i \
      -p $PORT:22 \
      -e ENABLE_AGENT=true \
      -e REIGNX_SERVER=host.docker.internal:50051 \
      -e NODE_ID=admtools-node-$(printf "%03d" $i) \
      reignx-admtools:latest

    echo "Started admtools-$i on port $PORT"
done

echo ""
echo "All containers started!"
echo "SSH: ssh root@localhost -p 222X (X = 3-12)"
```

### 4. 使用 Docker Compose

创建 `docker-compose.admtools.yaml`:

```yaml
version: '3.8'

services:
  admtools-1:
    image: reignx-admtools:latest
    container_name: admtools-1
    ports:
      - "2223:22"
    environment:
      ENABLE_AGENT: "true"
      REIGNX_SERVER: "host.docker.internal:50051"
      NODE_ID: "admtools-compose-001"
      LOG_LEVEL: "info"
    restart: unless-stopped

  admtools-2:
    image: reignx-admtools:latest
    container_name: admtools-2
    ports:
      - "2224:22"
    environment:
      ENABLE_AGENT: "true"
      REIGNX_SERVER: "host.docker.internal:50051"
      NODE_ID: "admtools-compose-002"
      LOG_LEVEL: "info"
    restart: unless-stopped

  admtools-3:
    image: reignx-admtools:latest
    container_name: admtools-3
    ports:
      - "2225:22"
    environment:
      ENABLE_AGENT: "true"
      REIGNX_SERVER: "host.docker.internal:50051"
      NODE_ID: "admtools-compose-003"
      LOG_LEVEL: "info"
    restart: unless-stopped
```

启动：

```bash
docker-compose -f docker-compose.admtools.yaml up -d

# 或使用 podman-compose
podman-compose -f docker-compose.admtools.yaml up -d
```

## 架构说明

### 镜像结构

```
reignx-admtools:latest
├─ Base: quay.io/openshift-psap-qe/admtools:latest
├─ SSH Server: OpenSSH Server
├─ ReignX Agent: /usr/local/bin/reignx-agent
├─ Users:
│  ├─ root (password: set via ROOT_PASSWORD)
│  └─ reignx (password: set via USER_PASSWORD, sudo access)
└─ Entrypoint: /entrypoint.sh
   ├─ Start SSH server
   ├─ Start ReignX Agent (if enabled)
   └─ Keep alive
```

### 多架构支持

镜像使用 manifest list 支持多个架构：

```bash
# 查看镜像架构
podman manifest inspect reignx-admtools:latest

# 拉取时自动选择匹配的架构
# macOS Apple Silicon (M1/M2/M3) → linux/arm64
# macOS Intel → linux/amd64
```

## 故障排查

### 查看容器日志

```bash
# 查看启动日志
podman logs admtools

# 实时跟踪日志
podman logs -f admtools

# 查看 SSH 日志
podman exec admtools tail -f /var/log/auth.log
```

### 验证 Agent 运行状态

```bash
# 进入容器
podman exec -it admtools bash

# 检查 Agent 进程
ps aux | grep reignx-agent

# 检查 Agent 日志
journalctl -u reignx-agent -f

# 手动测试 Agent
su - reignx
/usr/local/bin/reignx-agent --config /etc/reignx/agent.yaml
```

### SSH 连接问题

```bash
# 检查 SSH 服务状态
podman exec admtools ps aux | grep sshd

# 重启 SSH 服务
podman exec admtools pkill sshd
podman exec admtools /usr/sbin/sshd

# 测试 SSH 端口
nc -zv localhost 2222
```

### Agent 连接问题

```bash
# 测试到 ReignX Server 的连接
podman exec admtools nc -zv host.docker.internal 50051

# 检查 DNS 解析
podman exec admtools nslookup host.docker.internal

# 查看 Agent 配置
podman exec admtools cat /etc/reignx/agent.yaml
```

## 生产环境建议

### 1. 更改默认密码

```bash
# 运行容器后立即更改密码
podman exec admtools passwd root
podman exec admtools passwd reignx

# 或通过环境变量传递 (需修改 Containerfile)
```

### 2. 禁用密码认证

```bash
# 进入容器
podman exec -it admtools bash

# 修改 SSH 配置
sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# 重启 SSH
pkill sshd
/usr/sbin/sshd
```

### 3. 使用 TLS 证书

```bash
# 生成证书（使用 ReignX 提供的脚本）
cd /Users/liqcui/goproject/github.com/liqcui/reignx/certs
./generate-certs.sh

# 运行容器并启用 TLS
podman run -d --name admtools-secure \
  -p 2222:22 \
  -v $(pwd)/certs:/etc/reignx:ro \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=api.production.com:50051 \
  -e NODE_ID=admtools-prod-001 \
  -e TLS_ENABLED=true \
  reignx-admtools:latest
```

### 4. 资源限制

```bash
# 限制 CPU 和内存
podman run -d --name admtools \
  --cpus=2 \
  --memory=2g \
  --memory-swap=2g \
  -p 2222:22 \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=host.docker.internal:50051 \
  reignx-admtools:latest
```

## 常见用例

### 用例 1：本地开发测试

```bash
# 启动单个测试节点
podman run -d --name dev-node \
  -p 2222:22 \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=host.docker.internal:50051 \
  -e NODE_ID=dev-test-001 \
  -e LOG_LEVEL=debug \
  reignx-admtools:latest

# SSH 登录测试
ssh root@localhost -p 2222
```

### 用例 2：模拟多节点集群

```bash
# 使用脚本启动 20 个节点
for i in {1..20}; do
    podman run -d --name cluster-node-$i \
      -p $((2222 + i)):22 \
      -e ENABLE_AGENT=true \
      -e REIGNX_SERVER=host.docker.internal:50051 \
      -e NODE_ID=cluster-node-$(printf "%03d" $i) \
      reignx-admtools:latest
done
```

### 用例 3：CI/CD 集成测试

```bash
# 在 CI 管道中启动测试节点
docker run -d --name ci-test-node \
  -e ENABLE_AGENT=true \
  -e REIGNX_SERVER=${CI_REIGNX_SERVER} \
  -e NODE_ID=ci-test-${CI_JOB_ID} \
  reignx-admtools:latest

# 等待节点注册
sleep 10

# 运行集成测试
./run-integration-tests.sh

# 清理
docker rm -f ci-test-node
```

## 参考资源

- [OpenShift admtools 镜像](https://quay.io/repository/openshift-psap-qe/admtools)
- [ReignX 文档](../README.md)
- [ReignX Agent 配置](../reignx-agent/README.md)
- [多架构镜像构建](https://docs.docker.com/build/building/multi-platform/)
