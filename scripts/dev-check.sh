#!/bin/bash
#
# Copyright 2025 coze-dev Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


# Coze Studio 开发规范检查脚本
# 用法: ./scripts/dev-check.sh [--fix]

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Coze Studio 开发规范检查开始..."
echo

# 检查是否在项目根目录
if [[ ! -f "rush.json" ]]; then
    echo -e "${RED}❌ 错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

ERRORS=0
WARNINGS=0
FIX_MODE=false

if [[ "$1" == "--fix" ]]; then
    FIX_MODE=true
    echo -e "${YELLOW}🔧 修复模式已启用${NC}"
    echo
fi

# 1. IDL-First流程检查
echo "📋 1. IDL-First 流程检查"
echo "----------------------------------------"

# 检查IDL文件是否包含service定义
if grep -q "service.*WorkflowService" idl/workflow/workflow.thrift 2>/dev/null; then
    echo -e "${GREEN}✅ IDL文件包含service定义${NC}"
else
    echo -e "${RED}❌ IDL文件缺少service定义${NC}"
    echo "   请在idl/workflow/workflow.thrift中添加service定义"
    ERRORS=$((ERRORS + 1))
fi

# 检查是否有未生成的IDL更改
if [[ -n $(git status --porcelain idl/) ]]; then
    echo -e "${YELLOW}⚠️ IDL文件有未提交的更改，请确保重新生成代码${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

echo

# 2. 后端编译检查
echo "🔧 2. 后端编译检查"
echo "----------------------------------------"

cd backend
if go build ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 后端编译通过${NC}"
else
    echo -e "${RED}❌ 后端编译失败${NC}"
    echo "   运行以下命令查看详细错误："
    echo "   cd backend && go build ./..."
    ERRORS=$((ERRORS + 1))
fi
cd ..

echo

# 3. 前端Lint检查
echo "🎨 3. 前端Lint检查"
echo "----------------------------------------"

if $FIX_MODE; then
    echo "🔧 正在尝试自动修复Lint错误..."
    if rush lint --fix > /tmp/lint_output 2>&1; then
        echo -e "${GREEN}✅ 前端Lint检查通过（已自动修复）${NC}"
    else
        echo -e "${RED}❌ 前端Lint检查失败（部分错误无法自动修复）${NC}"
        tail -20 /tmp/lint_output
        ERRORS=$((ERRORS + 1))
    fi
else
    if rush lint > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 前端Lint检查通过${NC}"
    else
        echo -e "${RED}❌ 前端Lint检查失败${NC}"
        echo "   运行以下命令查看详细错误："
        echo "   rush lint"
        echo "   或运行自动修复："
        echo "   ./scripts/dev-check.sh --fix"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo

# 4. 代码质量检查
echo "⚡ 4. 代码质量检查"
echo "----------------------------------------"

# 检查硬编码IP地址
HARDCODE_IPS=$(grep -r "10\.10\.10\.208\|192\.168\." backend/ frontend/ --include="*.go" --include="*.ts" --include="*.tsx" 2>/dev/null || true)
if [[ -n "$HARDCODE_IPS" ]]; then
    echo -e "${YELLOW}⚠️ 发现硬编码IP地址:${NC}"
    echo "$HARDCODE_IPS" | head -5
    WARNINGS=$((WARNINGS + 1))
else
    echo -e "${GREEN}✅ 无硬编码IP地址${NC}"
fi

# 检查直接使用fetch的情况
DIRECT_FETCH=$(grep -r "fetch(" frontend/packages/ --include="*.ts" --include="*.tsx" 2>/dev/null || true)
if [[ -n "$DIRECT_FETCH" ]]; then
    echo -e "${YELLOW}⚠️ 发现直接使用fetch调用（建议使用生成的API客户端）:${NC}"
    echo "$DIRECT_FETCH" | head -3
    WARNINGS=$((WARNINGS + 1))
else
    echo -e "${GREEN}✅ 未发现直接fetch调用${NC}"
fi

# 检查中文字符串（简单检查）
CHINESE_STRINGS=$(grep -r "[\u4e00-\u9fff]" frontend/packages/ --include="*.tsx" --include="*.ts" 2>/dev/null | grep -v "I18n.t" || true)
if [[ -n "$CHINESE_STRINGS" ]]; then
    echo -e "${YELLOW}⚠️ 发现未国际化的中文字符串:${NC}"
    echo "$CHINESE_STRINGS" | head -3
    WARNINGS=$((WARNINGS + 1))
else
    echo -e "${GREEN}✅ 中文字符串国际化检查通过${NC}"
fi

echo

# 5. 架构规范检查
echo "🏗️ 5. 架构规范检查"
echo "----------------------------------------"

# 检查DDD目录结构
if [[ -d "backend/domain" && -d "backend/application" && -d "backend/api" ]]; then
    echo -e "${GREEN}✅ DDD分层架构目录结构正确${NC}"
else
    echo -e "${RED}❌ DDD分层架构目录结构不完整${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 检查前端包结构
MISSING_CONFIGS=0
for pkg_dir in frontend/packages/*/; do
    if [[ -f "$pkg_dir/package.json" ]]; then
        if [[ ! -f "$pkg_dir/tsconfig.json" ]]; then
            echo -e "${YELLOW}⚠️ 缺少tsconfig.json: $pkg_dir${NC}"
            MISSING_CONFIGS=$((MISSING_CONFIGS + 1))
        fi
        if [[ ! -f "$pkg_dir/eslint.config.js" ]]; then
            echo -e "${YELLOW}⚠️ 缺少eslint.config.js: $pkg_dir${NC}"
            MISSING_CONFIGS=$((MISSING_CONFIGS + 1))
        fi
    fi
done

if [[ $MISSING_CONFIGS -eq 0 ]]; then
    echo -e "${GREEN}✅ 前端包配置文件检查通过${NC}"
else
    echo -e "${YELLOW}⚠️ 发现 $MISSING_CONFIGS 个包配置文件问题${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

echo
echo "📊 检查结果总结"
echo "========================================"

if [[ $ERRORS -eq 0 && $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}🎉 所有检查项目通过！代码质量优秀。${NC}"
    exit 0
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}⚠️ 检查完成，发现 $WARNINGS 个警告项，建议修复。${NC}"
    exit 0
else
    echo -e "${RED}❌ 检查失败，发现 $ERRORS 个错误和 $WARNINGS 个警告。${NC}"
    echo
    echo "🔧 常用修复命令："
    echo "   # IDL代码重新生成"
    echo "   cd backend && hz update -idl ../idl/workflow/workflow.thrift"
    echo "   cd frontend/packages/arch/api-schema && npm run update"
    echo
    echo "   # 代码格式修复"
    echo "   rush lint --fix"
    echo "   cd backend && go fmt ./..."
    echo
    echo "   # 重新构建"
    echo "   rush build"
    echo "   cd backend && go build ./..."
    exit 1
fi