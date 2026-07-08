# 新增工作流节点完整开发脚本

> 本文档基于 card_selector 节点的成功开发经验，提供了一套完整的、可重复执行的开发流程。

## 📋 目录

1. [开发准备](#1-开发准备)
2. [后端开发](#2-后端开发)
3. [前端开发](#3-前端开发)
4. [系统注册](#4-系统注册)
5. [测试验证](#5-测试验证)
6. [常见问题](#6-常见问题)

## 1. 开发准备

### 1.1 环境检查

```bash
# 确保在项目根目录
cd /Users/dev/myproject/cursor/coze-studio

# 检查Git状态
git status

# 检查Rush环境
rush update
```

### 1.2 节点设计

在开始编码前，明确以下信息：

- **节点名称**：中文显示名称（如：卡片选择）
- **节点类型标识**：英文标识（如：card_selector）
- **节点ID**：数字ID（如：1001，使用未占用的大数字）
- **节点分类**：所属类别（如：utilities）
- **节点描述**：功能描述
- **输入参数**：节点需要的输入数据
- **输出参数**：节点产生的输出数据
- **图标URL**：节点图标地址

### 1.3 创建开发任务清单

创建ToDo List用于跟踪进度：
1. [ ] 后端节点类型定义
2. [ ] 后端节点实现
3. [ ] 后端节点注册
4. [ ] 前端节点类型定义
5. [ ] 前端节点启用
6. [ ] 前端节点组件实现
7. [ ] 前端节点注册
8. [ ] 测试验证

## 2. 后端开发

### 2.1 添加节点类型定义

**文件**: `/backend/domain/workflow/entity/node_meta.go`

#### 步骤 1：添加节点类型常量

```go
// 在 const 块中添加新节点类型
NodeTypeYourNode NodeType = "your_node_type"
```

#### 步骤 2：添加节点元信息

```go
// 在 NodeTypeMetas map 中添加配置
NodeTypeYourNode: {
    ID:           1002, // 使用下一个可用ID
    Key:          NodeTypeYourNode,
    DisplayKey:   "YourNode",
    Name:         "你的节点名称",
    Category:     "你的节点分类", // utilities, logic, input&output 等
    Desc:         "节点功能描述",
    Color:        "#4A90E2", // 节点颜色
    IconURL:      "https://example.com/icon.png", // 图标URL
    SupportBatch: false, // 是否支持批处理
    ExecutableMeta: ExecutableMeta{
        DefaultTimeoutMS: 30 * 1000, // 超时时间(毫秒)
        PreFillZero:      true,
        PostFillNil:      true,
    },
    EnUSName:        "Your Node", // 英文名称
    EnUSDescription: "Node description in English",
},
```

### 2.2 创建节点实现

**目录**: `/backend/domain/workflow/internal/nodes/yournode/`

#### 步骤 1：创建目录

```bash
mkdir -p /Users/dev/myproject/cursor/coze-studio/backend/domain/workflow/internal/nodes/yournode
```

#### 步骤 2：创建节点实现文件

**文件**: `your_node.go`

```go
/*
 * Copyright 2025 ynet-dev Authors
 * [License header...]
 */

package yournode

import (
    "context"
    "fmt"
    // 其他必要的导入
    
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity"
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/internal/canvas/convert"
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/internal/nodes"
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/internal/schema"
)

// 定义输入输出常量
const (
    InputKeyExample  = "example_input"
    OutputKeyResult  = "result"
)

// Config 实现 NodeAdaptor 和 NodeBuilder 接口
type Config struct {
    // 节点特定的配置字段
    ConfigField string `json:"config_field,omitempty"`
}

// Adapt 实现 NodeAdaptor 接口
func (c *Config) Adapt(ctx context.Context, n *vo.Node, opts ...nodes.AdaptOption) (*schema.NodeSchema, error) {
    ns := &schema.NodeSchema{
        Key:     vo.NodeKey(n.ID),
        Type:    entity.NodeTypeYourNode,
        Name:    n.Data.Meta.Title,
        Configs: c,
    }

    // 设置输入字段类型和映射信息
    if err := convert.SetInputsForNodeSchema(n, ns); err != nil {
        return nil, err
    }

    // 设置输出字段类型信息
    if err := convert.SetOutputTypesForNodeSchema(n, ns); err != nil {
        return nil, err
    }

    return ns, nil
}

// Build 实现 NodeBuilder 接口
func (c *Config) Build(ctx context.Context, ns *schema.NodeSchema, opts ...schema.BuildOption) (any, error) {
    return &YourNode{
        configField: c.ConfigField,
    }, nil
}

// YourNode 是实际的节点实现
type YourNode struct {
    configField string
}

// Invoke 实现 InvokableNode 接口
func (yn *YourNode) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // 获取输入参数
    exampleInput, ok := input[InputKeyExample].(string)
    if !ok {
        return nil, fmt.Errorf("example_input is required and must be a string")
    }

    // 实现节点逻辑
    result := fmt.Sprintf("Processed: %s", exampleInput)

    // 返回结果
    return map[string]any{
        OutputKeyResult: result,
    }, nil
}
```

### 2.3 注册节点适配器

**文件**: `/backend/domain/workflow/internal/canvas/adaptor/to_schema.go`

#### 步骤 1：添加导入

```go
import (
    // ... 其他导入
    "github.com/ynet-dev/ynet-studio/backend/domain/workflow/internal/nodes/yournode"
)
```

#### 步骤 2：注册适配器

在 `RegisterAllNodeAdaptors` 函数中添加：

```go
nodes.RegisterNodeAdaptor(entity.NodeTypeYourNode, func() nodes.NodeAdaptor {
    return &yournode.Config{}
})
```

## 3. 前端开发

### 3.1 添加节点类型定义

**文件**: `/frontend/packages/workflow/base/src/types/node-type.ts`

```typescript
export enum StandardNodeType {
  // ... 其他节点类型
  
  // Your Node
  YourNode = '1002', // 使用对应的ID
}
```

### 3.2 启用节点类型

**文件**: `/frontend/packages/workflow/adapter/base/src/utils/get-enabled-node-types.ts`

```typescript
const nodesMap = {
  // ... 其他节点
  [StandardNodeType.YourNode]: true,
};
```

### 3.3 创建前端节点实现

**目录**: `/frontend/packages/workflow/playground/src/node-registries/your-node/`

#### 步骤 1：创建目录

```bash
mkdir -p /Users/dev/myproject/cursor/coze-studio/frontend/packages/workflow/playground/src/node-registries/your-node
```

#### 步骤 2：创建基础文件

**constants.ts**：
```typescript
import { nanoid } from 'nanoid';
import { ViewVariableType } from '@coze-workflow/variable';

// 路径定义
export const INPUT_PATH = 'inputParameters';
export const YOUR_NODE_CONFIG_PATH = 'yourNodeConfig';
export const OUTPUT_PATH = 'outputs';

// 默认输出
export const DEFAULT_OUTPUTS = [
  {
    key: nanoid(),
    name: 'result',
    type: ViewVariableType.String,
  },
];

// 默认输入
export const DEFAULT_INPUTS = [
  { name: 'example_input' }
];
```

**types.ts**：
```typescript
import type { OutputTreeMeta, Parameter } from '@coze-workflow/base';

export interface YourNodeConfig {
  configField?: string;
}

export interface FormData {
  inputParameters: Parameter[];
  yourNodeConfig: YourNodeConfig;
  outputs: OutputTreeMeta[];
}
```

**data-transformer.ts**：
```typescript
import { type NodeData } from '@coze-workflow/base';
import { isEmpty } from '@coze-arch/utils';
import { type FormData } from './types';
import { DEFAULT_INPUTS, DEFAULT_OUTPUTS } from './constants';

export function transformOnInit(data: NodeData): FormData {
  return {
    inputParameters: data?.inputParameters || DEFAULT_INPUTS,
    yourNodeConfig: {
      configField: data?.yourNodeConfig?.configField || '',
    },
    outputs: data?.outputs || DEFAULT_OUTPUTS,
  };
}

export function transformOnSubmit(data: FormData): NodeData {
  return {
    inputParameters: data.inputParameters,
    yourNodeConfig: {
      configField: data.yourNodeConfig?.configField || '',
    },
    outputs: isEmpty(data.outputs) ? DEFAULT_OUTPUTS : data.outputs,
  };
}
```

**node-test.ts**：
```typescript
import { FlowNodeFormData } from '@flowgram-adapter/free-layout-editor';
import { type NodeTestMeta, generateParametersToProperties } from '@/test-run-kit';

export const test: NodeTestMeta = {
  generateFormInputProperties(node) {
    const formData = node
      .getData(FlowNodeFormData)
      .formModel.getFormItemValueByPath('/');
    const parameters = formData?.inputParameters;

    return generateParametersToProperties(parameters, { node });
  },
};
```

#### 步骤 3：创建UI组件

**components/your-node-field.tsx**：
```typescript
import React from 'react';
import { useField, withField } from '@/form';
import type { YourNodeConfig } from '../types';

interface YourNodeFieldProps {
  tooltip?: string;
}

export const YourNodeField = withField(({ tooltip }: YourNodeFieldProps) => {
  const { value, onChange, errors } = useField<YourNodeConfig>();

  const handleConfigChange = (field: keyof YourNodeConfig) => 
    (fieldValue: string) => {
      onChange({
        ...value,
        [field]: fieldValue,
      });
    };

  const feedbackText = errors?.[0]?.message || '';

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: 16 }}>
        <div style={{ 
          fontSize: '12px', 
          fontWeight: 600, 
          marginBottom: 8, 
          color: 'var(--semi-color-text-0)' 
        }}>
          配置字段
        </div>
        <input
          placeholder="输入配置值..."
          value={value?.configField || ''}
          onChange={(e) => handleConfigChange('configField')(e.target.value)}
          style={{
            width: '100%',
            padding: '8px 12px',
            border: '1px solid var(--semi-color-border)',
            borderRadius: '6px',
            fontSize: '14px',
          }}
        />
      </div>

      {feedbackText && (
        <div style={{ 
          color: 'var(--semi-color-danger)', 
          fontSize: '12px', 
          marginTop: 8 
        }}>
          {feedbackText}
        </div>
      )}
    </div>
  );
});
```

**form.tsx**：
```typescript
import React from 'react';
import { I18n } from '@coze-arch/i18n';
import { NodeConfigForm } from '@/node-registries/common/components';
import { InputsParametersField, OutputsField } from '../common/fields';
import { YourNodeField } from './components/your-node-field';
import { INPUT_PATH, YOUR_NODE_CONFIG_PATH, OUTPUT_PATH } from './constants';

export function FormRender() {
  return (
    <NodeConfigForm>
      <InputsParametersField
        name={INPUT_PATH}
        title="输入参数"
        tooltip="配置节点的输入参数"
        id="your-node-inputs"
      />

      <YourNodeField
        name={YOUR_NODE_CONFIG_PATH}
        title="节点配置"
        tooltip="配置节点的特定参数"
        id="your-node-config"
      />

      <OutputsField
        title="输出参数"
        tooltip="节点的输出结果"
        id="your-node-outputs"
        name={OUTPUT_PATH}
        topLevelReadonly={true}
        customReadonly
      />
    </NodeConfigForm>
  );
}
```

**form-meta.tsx**：
```typescript
import {
  ValidateTrigger,
  type FormMetaV2,
} from '@flowgram-adapter/free-layout-editor';

import { nodeMetaValidate } from '@/nodes-v2/materials/node-meta-validate';
import {
  fireNodeTitleChange,
  provideNodeOutputVariablesEffect,
} from '@/node-registries/common/effects';

import { outputTreeMetaValidator } from '../common/fields/outputs';
import { type FormData } from './types';
import { FormRender } from './form';
import { transformOnInit, transformOnSubmit } from './data-transformer';
import { YOUR_NODE_CONFIG_PATH, OUTPUT_PATH } from './constants';

export const YOUR_NODE_FORM_META: FormMetaV2<FormData> = {
  render: () => <FormRender />,
  validateTrigger: ValidateTrigger.onChange,
  validate: {
    nodeMeta: nodeMetaValidate,
    [OUTPUT_PATH]: outputTreeMetaValidator,
  },
  effect: {
    nodeMeta: fireNodeTitleChange,
    [OUTPUT_PATH]: provideNodeOutputVariablesEffect,
  },
  formatOnInit: transformOnInit,
  formatOnSubmit: transformOnSubmit,
};
```

**node-content.tsx**：
```typescript
import React from 'react';
import { InputParameters, Outputs } from '../common/components';

export function YourNodeContent() {
  return (
    <>
      <InputParameters />
      <Outputs />
    </>
  );
}
```

**node-registry.ts**：
```typescript
import {
  DEFAULT_NODE_META_PATH,
  DEFAULT_OUTPUTS_PATH,
} from '@coze-workflow/nodes';
import {
  StandardNodeType,
  type WorkflowNodeRegistry,
} from '@coze-workflow/base';

import { type NodeTestMeta } from '@/test-run-kit';
import { test } from './node-test';
import { YOUR_NODE_FORM_META } from './form-meta';
import { INPUT_PATH } from './constants';

export const YOUR_NODE_REGISTRY: WorkflowNodeRegistry<NodeTestMeta> = {
  type: StandardNodeType.YourNode,
  meta: {
    nodeDTOType: StandardNodeType.YourNode,
    size: { width: 360, height: 130 },
    test,
    nodeMetaPath: DEFAULT_NODE_META_PATH,
    outputsPath: DEFAULT_OUTPUTS_PATH,
    inputParametersPath: INPUT_PATH,
    enableCopilotGenerateTestNodeForm: false,
  },
  formMeta: YOUR_NODE_FORM_META,
};
```

**index.ts**：
```typescript
export { YOUR_NODE_REGISTRY } from './node-registry';
export { YourNodeContent } from './node-content';
export { YourNodeField } from './components/your-node-field';
export * from './types';
export * from './constants';
```

## 4. 系统注册

### 4.1 注册到前端节点列表

**文件**: `/frontend/packages/workflow/playground/src/node-registries/index.ts`

```typescript
export { YOUR_NODE_REGISTRY } from './your-node';
```

### 4.2 添加到节点常量列表

**文件**: `/frontend/packages/workflow/playground/src/nodes-v2/constants.ts`

```typescript
import { YOUR_NODE_REGISTRY } from '@/node-registries/your-node';

export const NODES_V2 = [
  // ... 其他节点
  YOUR_NODE_REGISTRY,
];
```

## 5. 测试验证

### 5.1 编译测试

```bash
# 后端编译测试
cd /Users/dev/myproject/cursor/coze-studio/backend
go build -o test-server .

# 前端构建测试  
cd /Users/dev/myproject/cursor/coze-studio/frontend/packages/workflow/playground
npm run build

# 或者整体构建测试
cd /Users/dev/myproject/cursor/coze-studio
rush build --to @coze-studio/app
```

### 5.2 运行时测试

```bash
# 启动后端服务
cd /Users/dev/myproject/cursor/coze-studio
make server

# 启动前端服务
cd /Users/dev/myproject/cursor/coze-studio/frontend/apps/coze-studio  
npm run dev
```

### 5.3 功能测试清单

- [ ] 节点在左侧面板的对应分类中可见
- [ ] 可以将节点拖拽到画布
- [ ] 节点显示正确的名称和图标
- [ ] 可以打开节点配置面板
- [ ] 输入参数配置功能正常
- [ ] 节点特有配置功能正常
- [ ] 输出参数配置功能正常
- [ ] 保存配置后数据正确存储
- [ ] 可以连接其他节点
- [ ] 支持单节点试运行
- [ ] 工作流执行时节点逻辑正常

## 6. 常见问题与解决方案

> ⚠️ **重要提醒**: 本节基于实际开发过程中遇到的真实问题，务必仔细阅读！

### 6.1 前端模块依赖错误（高频问题）

这是开发过程中最常遇到的问题！以下是完整的错误现象和解决方案：

#### ❌ 错误现象
当运行 `npm run dev` 时，控制台会显示类似错误：

```bash
error   Compile error: 
Failed to compile, check the errors for troubleshooting.
File: /Users/.../card-selector/data-transformer.ts:1:1
  × Module not found: Can't resolve '@coze-arch/utils'
  × Module not found: Can't resolve '@semi-design/ui'  
  × Module not found: Can't resolve '@semi-design/icons'
  × Module not found: Can't resolve '@/node-registries/common/hooks'
```

#### ✅ 解决方案

**1. @coze-arch/utils 依赖问题**

❌ **错误写法**:
```typescript
import { isEmpty } from '@coze-arch/utils';

export function transformOnSubmit(data: FormData): NodeData {
  return {
    outputs: isEmpty(data.outputs) ? DEFAULT_OUTPUTS : data.outputs,
  };
}
```

✅ **正确写法**:
```typescript
// 移除 @coze-arch/utils 导入
export function transformOnSubmit(data: FormData): NodeData {
  return {
    outputs: (!data.outputs || data.outputs.length === 0) ? DEFAULT_OUTPUTS : data.outputs,
  };
}
```

**2. @semi-design/ui 和 @semi-design/icons 依赖问题**

❌ **错误写法**:
```typescript
import { Typography, Space } from '@semi-design/ui';
import { IconCard } from '@semi-design/icons';

const { Text } = Typography;

export function YourNodeContent() {
  return (
    <Space>
      <IconCard />
      <Text>内容</Text>
    </Space>
  );
}
```

✅ **正确写法**:
```typescript
// 移除 Semi Design 导入，使用简化实现
export function YourNodeContent() {
  return (
    <>
      <InputParameters />
      <Outputs />
    </>
  );
}
```

**3. common/hooks 模块缺失问题**

❌ **错误写法**:
```typescript
import { useFormData } from '@/node-registries/common/hooks';

export function YourNodeContent() {
  const formData = useFormData<FormData>();
  // 使用 formData...
}
```

✅ **正确写法**:
```typescript
// 移除不存在的 hooks，简化组件
export function YourNodeContent() {
  return (
    <>
      <InputParameters />
      <Outputs />
    </>
  );
}
```

#### 🔍 快速诊断方法

1. **检测命令**:
```bash
cd frontend/apps/coze-studio
npm run dev
# 查看控制台输出的 "Module not found" 错误
```

2. **修复验证**:
```bash
cd frontend/apps/coze-studio  
npm run build
# 如果构建成功，说明依赖问题已解决
```

### 6.2 Go编译错误

**问题**: Go导入错误
```
package xxx is not in GOROOT
```

**解决方案**: 检查import路径是否正确，确保包名与目录结构匹配。

**问题**: TypeScript类型错误
```
Property 'xxx' does not exist on type 'yyy'
```

**解决方案**: 检查类型定义是否正确，确保导入了必要的类型。

### 6.3 运行时错误

**问题**: 节点不显示在面板中

**解决方案**: 
1. 检查节点是否在 `get-enabled-node-types.ts` 中启用
2. 检查节点是否注册到 `constants.ts` 
3. 检查节点类型枚举是否正确定义

**问题**: 配置保存失败

**解决方案**:
1. 检查 `data-transformer.ts` 中的转换逻辑
2. 检查表单验证规则
3. 检查字段路径是否正确

### 6.4 最佳实践

1. **命名规范**
   - 后端包名使用小写字母
   - 前端组件使用PascalCase
   - 常量使用UPPER_SNAKE_CASE

2. **错误处理**
   - 后端要有完整的错误处理
   - 前端要有用户友好的错误提示

3. **类型安全**
   - 使用TypeScript严格模式
   - 定义完整的接口类型

4. **文档注释**
   - 重要函数要有注释说明
   - 复杂逻辑要有解释

### 6.5 调试技巧

1. **后端调试**
   ```bash
   # 查看日志
   make server
   
   # 使用Go调试器
   dlv debug
   ```

2. **前端调试**
   ```bash
   # 开发模式启动
   cd frontend/apps/coze-studio
   npm run dev
   
   # 使用浏览器开发者工具
   # 检查Console和Network面板
   ```

3. **依赖问题排查**
   ```bash
   # 编译测试（快速）
   cd frontend/packages/workflow/playground
   npm run build
   
   # 或者编译整个前端应用（完整）
   cd frontend/apps/coze-studio  
   npm run build
   
   # 检查模块导入是否正确
   grep -r "Module not found" node_modules/.cache/ || echo "No module errors"
   ```

4. **常见修复步骤**
   ```bash
   # 1. 清理依赖
   rm -rf node_modules package-lock.json
   npm install
   
   # 2. Rush更新（monorepo项目）
   rush update
   
   # 3. 重新构建
   rush build --to @coze-studio/app
   ```

## 7. 避坑指南 ⚠️

> 基于真实开发经验总结，强烈建议开发前阅读！

### 7.1 依赖使用原则

❌ **禁止使用的模块** (会导致编译错误):
```typescript
// 这些导入会导致 "Module not found" 错误
import { isEmpty } from '@coze-arch/utils';          // ❌ 不存在
import { Typography, Space } from '@semi-design/ui'; // ❌ 不可用
import { IconCard } from '@semi-design/icons';       // ❌ 不可用
import { useFormData } from '@/node-registries/common/hooks'; // ❌ 目录不存在
```

✅ **推荐使用的模块**:
```typescript
// 这些是安全的导入
import React from 'react';                                    // ✅ 基础React
import { useField, withField } from '@/form';                // ✅ 表单系统
import { InputParameters, Outputs } from '../common/components'; // ✅ 通用组件
import { type NodeData } from '@coze-workflow/base';         // ✅ 基础类型
```

### 7.2 开发顺序建议

1. **先写后端，后写前端** - 确保数据流设计合理
2. **先简化实现，再优化** - 避免一开始就使用复杂依赖
3. **频繁编译测试** - 每完成一个文件就测试编译
4. **渐进式开发** - 先实现基本功能，再添加高级特性

### 7.3 代码模板使用技巧

1. **复制现有节点代码** - 从类似的现有节点开始
2. **批量替换节点名称** - 使用编辑器的查找替换功能
3. **保持文件结构一致** - 严格按照模板的目录结构

### 7.4 编译错误应对策略

遇到编译错误时的处理顺序：

1. **立即停止添加新功能** - 专注解决当前错误
2. **查看完整错误信息** - 不要只看第一个错误
3. **先修复依赖问题** - 模块导入错误优先处理
4. **逐个文件验证** - 确保每个文件都能单独通过类型检查

### 7.5 测试验证流程

```bash
# 标准验证流程，每个步骤都必须通过
cd /Users/dev/myproject/cursor/coze-studio

# 1. 后端编译测试
cd backend && go build -o test-server .

# 2. 前端编译测试  
cd frontend/apps/coze-studio && npm run build

# 3. 前端开发模式测试
npm run dev
# 访问 http://localhost:8080 检查控制台是否有错误

# 4. 功能测试
# 在浏览器中测试节点是否出现在面板中
```

## 8. 模板文件清单

使用本脚本开发节点时，需要创建以下文件：

### 后端文件 (4个)
1. `backend/domain/workflow/entity/node_meta.go` - 修改
2. `backend/domain/workflow/internal/nodes/yournode/your_node.go` - 新建
3. `backend/domain/workflow/internal/canvas/adaptor/to_schema.go` - 修改

### 前端文件 (11个)
1. `frontend/packages/workflow/base/src/types/node-type.ts` - 修改
2. `frontend/packages/workflow/adapter/base/src/utils/get-enabled-node-types.ts` - 修改  
3. `frontend/packages/workflow/playground/src/node-registries/your-node/constants.ts` - 新建
4. `frontend/packages/workflow/playground/src/node-registries/your-node/types.ts` - 新建
5. `frontend/packages/workflow/playground/src/node-registries/your-node/data-transformer.ts` - 新建
6. `frontend/packages/workflow/playground/src/node-registries/your-node/node-test.ts` - 新建
7. `frontend/packages/workflow/playground/src/node-registries/your-node/components/your-node-field.tsx` - 新建
8. `frontend/packages/workflow/playground/src/node-registries/your-node/form.tsx` - 新建
9. `frontend/packages/workflow/playground/src/node-registries/your-node/form-meta.tsx` - 新建
10. `frontend/packages/workflow/playground/src/node-registries/your-node/node-content.tsx` - 新建
11. `frontend/packages/workflow/playground/src/node-registries/your-node/node-registry.ts` - 新建
12. `frontend/packages/workflow/playground/src/node-registries/your-node/index.ts` - 新建
13. `frontend/packages/workflow/playground/src/node-registries/index.ts` - 修改
14. `frontend/packages/workflow/playground/src/nodes-v2/constants.ts` - 修改

## 9. 检查清单

开发完成后，使用此清单验证：

### ✅ 基础功能检查
- [ ] 所有必要文件已创建
- [ ] 后端编译测试通过 (`go build` 成功)
- [ ] 前端编译测试通过 (`npm run build` 成功)
- [ ] 前端开发模式启动成功 (`npm run dev` 无错误)

### ✅ UI功能检查  
- [ ] 节点在左侧面板的正确分类中显示
- [ ] 节点图标和名称显示正确
- [ ] 可以将节点拖拽到画布上
- [ ] 可以打开节点配置面板
- [ ] 输入参数配置功能正常
- [ ] 节点特有配置功能正常
- [ ] 输出参数配置功能正常

### ✅ 数据流检查
- [ ] 配置数据保存正常
- [ ] 刷新页面后配置数据恢复正常
- [ ] 可以连接到其他节点
- [ ] 支持单节点试运行（如果适用）
- [ ] 工作流执行时节点逻辑正常

### ✅ 代码质量检查
- [ ] 代码符合团队规范
- [ ] 添加了必要的注释
- [ ] 错误处理完整
- [ ] 没有使用被禁止的依赖模块
- [ ] 遵循了文档中的最佳实践

### ⚠️ 常见遗漏项目
- [ ] 检查控制台是否有 React 警告
- [ ] 检查是否有 TypeScript 类型错误
- [ ] 验证所有文件的导入路径正确
- [ ] 确认节点ID没有与现有节点冲突

## 🎉 总结

遵循此脚本，你可以系统性地开发出符合YNET Studio架构规范的新工作流节点。每个步骤都经过验证，确保开发过程的可靠性和一致性。

### 🔑 成功的关键
1. **严格按照步骤执行** - 不要跳过任何环节
2. **频繁测试验证** - 每完成一部分就编译测试
3. **避免使用禁止的依赖** - 参考避坑指南
4. **遇到问题及时查阅** - 使用文档中的问题解决方案

记住：**先理解现有代码模式，再复制成功的实现** 是最佳的开发策略！

### 📚 相关资源
- 本开发脚本文档：完整的step-by-step指南
- 项目README：项目整体架构和环境设置
- 现有节点代码：最佳的学习参考模板

Happy Coding! 🚀