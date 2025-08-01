# CreateSpace API 测试文档

## API 概述

新增了创建空间的API接口，允许用户创建新的工作空间。

### 接口信息

- **URL**: `POST /api/playground_api/space/create`
- **认证**: 需要用户登录（通过session_key）

### 请求参数

```json
{
  "name": "我的新空间",           // 必需：空间名称
  "description": "空间描述",      // 可选：空间描述
  "icon_uri": "icon/path.png"   // 可选：图标URI
}
```

### 响应格式

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "id": "7532755646102372352",
    "name": "我的新空间",
    "description": "空间描述", 
    "icon_url": "https://example.com/icon/path.png",
    "app_ids": []
  }
}
```

## 测试用例

### 使用curl测试

```bash
curl 'http://localhost:8080/api/playground_api/space/create' \
  -H 'Accept: application/json, text/plain, */*' \
  -H 'Accept-Language: zh-CN,zh;q=0.9' \
  -H 'Content-Type: application/json' \
  -b 'session_key=你的session_key' \
  -H 'Origin: http://localhost:8080' \
  --data-raw '{
    "name": "我的测试空间",
    "description": "这是一个测试空间",
    "icon_uri": ""
  }'
```

### 成功响应示例

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "id": "7533076982897049600",
    "name": "我的测试空间",
    "description": "这是一个测试空间",
    "icon_url": "",
    "app_ids": []
  }
}
```

### 错误响应示例

#### 名称为空
```json
{
  "code": 400,
  "msg": "space name is required"
}
```

#### 未登录
```json
{
  "code": 401,
  "msg": "authentication failed"
}
```

## 实现细节

### 数据库操作
1. 生成新的空间ID
2. 在`space`表中创建空间记录
3. 在`space_user`表中添加创建者为空间所有者（role_type=1）

### 权限角色
- **owner (1)**: 空间所有者
- **admin (2)**: 空间管理员  
- **member (3)**: 空间成员

创建空间时，创建者自动成为空间所有者。

## 相关API

配合使用的其他空间API：

- `POST /api/playground_api/space/list` - 获取空间列表
- 后续可扩展：
  - `PUT /api/playground_api/space/update` - 更新空间信息
  - `DELETE /api/playground_api/space/delete` - 删除空间
  - `POST /api/playground_api/space/members` - 管理空间成员