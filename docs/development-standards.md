# 开发规范与最佳实践

## 架构设计原则

### 领域驱动设计 (DDD)

```
backend/
├── api/           # HTTP接口层 - 请求绑定和响应格式化
├── application/   # 应用服务层 - 业务流程编排，事务管理
├── domain/        # 领域层 - 纯业务逻辑，无外部依赖
├── infra/         # 基础设施层 - 数据持久化，外部服务调用
├── crossdomain/   # 跨域服务 - 通用功能
└── types/         # 共享类型定义
```

**关键原则**：
- 单一职责：每层只处理自己的职责
- 依赖倒置：上层依赖抽象接口，不依赖具体实现
- 层级调用：API → Application → Domain → Infra（不能跨层调用）

### Handler 规则

Handler 只做请求绑定，**不写业务逻辑**：

```go
// 正确
func CreateSpace(ctx context.Context, c *app.RequestContext) {
    var req space.CreateSpaceRequest
    c.BindAndValidate(&req)
    resp, err := application.GetSpaceService().CreateSpace(ctx, &req)
    c.JSON(consts.StatusOK, resp)
}

// 错误 - 不要在 Handler 中写业务逻辑
```

---

## 环境要求

- **Go**: >= 1.24
- **Node.js**: >= 22
- **pnpm**: 8.15.8
- **Docker**: 容器化开发

## 开发命令

### 日常开发

```bash
# 启动中间件（MySQL, Redis, ES 等）
make middleware

# 启动后端
make server

# 启动前端
cd frontend/apps/coze-studio && npm run dev

# 完整环境
make debug
```

### 构建和测试

```bash
# 后端
cd backend && go test ./...
cd backend && go build ./...

# 前端
rush install    # 安装依赖
rush build      # 构建所有包
rush test       # 运行测试
rush lint       # 代码检查
```

### 数据库

```bash
make sync_db     # 同步 schema 到数据库
make dump_db     # 导出数据库 schema
make atlas-hash  # 重新哈希迁移文件
```

---

## 代码规范

### 后端 Go
- 强制 `gofmt` 格式化
- 统一错误处理模式（`backend/types/errno`）
- 核心业务逻辑 80%+ 测试覆盖率

### 前端 TypeScript
- React 18 + TypeScript + Tailwind CSS
- 状态管理：Zustand
- 测试框架：Vitest
- 包管理：Rush.js monorepo

### IDL 接口设计
- 服务方法 PascalCase
- 响应固定 `253: code`, `254: msg`
- 大整数加 `api.js_conv='true'`

---

## 前端 Monorepo 结构

```
frontend/
├── apps/coze-studio/      # 主应用 (Level 4)
├── packages/arch/         # 核心架构 (Level 1)
├── packages/components/   # UI 组件
├── packages/common/       # 共享工具 (Level 2)
├── packages/workflow/     # 工作流 (Level 3)
└── config/                # 共享配置
```

**重要**：
- `@coze-arch/bot-api` 是核心内部 API，**不要修改**
- 使用 `@coze-studio/api-schema` 做扩展

---

## 常用组件注意事项

### @coze-arch/coze-design

```typescript
// Input 的 onChange 直接接收 value，不是 event
<Input onChange={(value) => setValue(value)} />  // 正确
<Input onChange={(e) => setValue(e.target.value)} />  // 错误
```

---

## 配置文件索引

| 文件 | 用途 |
|------|------|
| `rush.json` | Monorepo 包定义 |
| `api.config.js` | IDL 到 TypeScript 生成配置 |
| `.hz` | 后端代码生成配置 |
| `docker-compose.yml` | 完整服务栈 |
| `Makefile` | 开发工作流命令 |
| `bin/.env.debug` | 开发环境变量 |
