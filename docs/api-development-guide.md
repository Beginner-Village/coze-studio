# API 开发完整指南

## 概述

Coze Studio 使用 Thrift IDL 定义 API 契约，通过 Hz 工具（后端）和 idl2ts（前端）生成类型安全的代码。

---

## 完整开发流程

### 阶段一：Thrift IDL 定义

#### 1. 创建 IDL 文件

文件位置：`/idl/[module_name]/[module_name].thrift`

```thrift
namespace go test_management

struct TestItem {
    1: required i64 id
    2: required string title
    3: optional string description
    4: required i32 status
    5: required i64 created_at
}

struct CreateTestRequest {
    1: required string title (api.body="title")
    2: optional string description (api.body="description")
}

struct CreateTestResponse {
    253: required i32 code
    254: required string msg
    1: required TestItem data
}

service TestManagementService {
    CreateTestResponse CreateTest(1: CreateTestRequest req) (api.post="/api/test/create")
    GetTestListResponse GetTestList(1: GetTestListRequest req) (api.get="/api/test/list")
}
```

#### IDL 关键规则
- 响应码字段固定位置：`253: required i32 code`、`254: required string msg`
- 路径参数：`(api.path="id")`
- 大整数防精度丢失：`(api.js_conv='true', agw.js_conv="str")`

### 阶段二：前端代码生成

```bash
# 1. 更新配置
# 文件：frontend/packages/arch/api-schema/api.config.js
# 添加：test_management: './idl/test_management.thrift'

# 2. 生成 TypeScript 代码
cd frontend/packages/arch/api-schema
npm run update

# 3. 检查导出（src/index.ts）
# export * as test_management from './idl/test_management';
```

### 阶段三：后端代码生成

```bash
# 1. 检查 INSERT_POINT 格式（关键！）
# backend/api/router/register.go 中必须是：
# //INSERT_POINT: DO NOT DELETE THIS LINE!
# 注意：双斜杠和 INSERT_POINT 之间不能有空格！

# 2. 生成代码
cd backend
hz update -idl ../idl/test_management/test_management.thrift

# 生成的文件：
# backend/api/model/test_management/test_management.go
# backend/api/handler/test_management/test_management_service.go
# backend/api/router/test_management/test_management.go
```

### 阶段四：实现业务逻辑

**核心原则：不要在 Handler 中写业务逻辑**

```go
// Handler 只做请求绑定和响应
func CreateTest(ctx context.Context, c *app.RequestContext) {
    var req test_management.CreateTestRequest
    err := c.BindAndValidate(&req)
    if err != nil {
        c.String(consts.StatusBadRequest, err.Error())
        return
    }

    // 调用 Application 层
    resp, err := application.TestService.Create(ctx, &req)
    if err != nil {
        c.String(consts.StatusInternalServerError, err.Error())
        return
    }
    c.JSON(consts.StatusOK, resp)
}
```

### 阶段五：前端调用

```typescript
import { test_management } from '@coze-studio/api-schema';  // 注意下划线

const result = await test_management.CreateTest({
    title: '测试',
    description: '描述'
});
```

---

## 常见问题

### Hz 工具 INSERT_POINT 错误
`insert-point '//INSERT_POINT:...' not found` → 检查 `register.go`，双斜杠后不能有空格。

### 前端 API 导入名错误
`Cannot read properties of undefined` → 使用下划线命名：`test_management`，不是 `testManagement`。

### 成功响应进入 catch
API 返回 200 但进入 catch → 检查 `error.code === '200' || error.code === 200`。

### 大整数精度丢失
JS 只能安全表示 -(2^53-1) 到 2^53-1 → IDL 中加 `api.js_conv='true'`，前端不要 `parseInt`。

### Hz 路由参数格式
Hz 生成 `{param}` 但 Hertz 需要 `:param` → 手动修改路由文件。

---

## 开发检查清单

### IDL 阶段
- [ ] IDL 文件位置正确
- [ ] 响应码字段 253/254
- [ ] 大整数字段加 js_conv

### 代码生成
- [ ] INSERT_POINT 格式正确
- [ ] 前端 api.config.js 更新
- [ ] index.ts 导出配置

### 实现
- [ ] Handler 只做请求绑定
- [ ] 业务逻辑在 Application 层
- [ ] 前端正确处理错误

### 测试
- [ ] curl 测试后端 API
- [ ] 前端页面联调
