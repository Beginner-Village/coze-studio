#!/bin/bash

# ChatFlow 数据库表同步脚本
# 使用docker/.env中的数据库配置连接远程数据库

echo "=== ChatFlow 数据库表同步脚本 ==="
echo "目标数据库: 10.10.10.224:3306"
echo "数据库名: opencoze"
echo ""

# 从.env文件读取数据库配置
MYSQL_HOST="10.10.10.224"
MYSQL_PORT="3306"
MYSQL_USER="coze"
MYSQL_PASSWORD="coze123"
MYSQL_DATABASE="opencoze"

# 检查MySQL客户端是否安装
if ! command -v mysql &> /dev/null; then
    echo "❌ 错误: 未找到mysql客户端，请先安装MySQL客户端"
    echo "macOS: brew install mysql-client"
    echo "Ubuntu: sudo apt-get install mysql-client"
    exit 1
fi

# 检查SQL脚本文件是否存在
if [ ! -f "sync_chatflow_tables.sql" ]; then
    echo "❌ 错误: 未找到 sync_chatflow_tables.sql 文件"
    exit 1
fi

echo "🔍 正在测试数据库连接..."

# 测试数据库连接
if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1;" "$MYSQL_DATABASE" &>/dev/null; then
    echo "✅ 数据库连接成功"
else
    echo "❌ 数据库连接失败，请检查配置:"
    echo "   - 主机: $MYSQL_HOST:$MYSQL_PORT"
    echo "   - 用户: $MYSQL_USER"
    echo "   - 数据库: $MYSQL_DATABASE"
    echo "   - 请确认网络连接和数据库服务状态"
    exit 1
fi

echo ""
echo "🚀 开始执行ChatFlow表同步..."
echo ""

# 执行SQL脚本
if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" < sync_chatflow_tables.sql; then
    echo ""
    echo "✅ ChatFlow表同步完成！"
    echo ""
    echo "已创建的表:"
    echo "  - app_conversation_template_draft   (应用对话模板草稿表)"
    echo "  - app_conversation_template_online  (应用对话模板在线表)"
    echo "  - app_dynamic_conversation_draft    (应用动态对话草稿表)"
    echo "  - app_dynamic_conversation_online   (应用动态对话在线表)" 
    echo "  - app_static_conversation_draft     (应用静态对话草稿表)"
    echo "  - app_static_conversation_online    (应用静态对话在线表)"
    echo "  - chat_flow_role_config             (ChatFlow角色配置表)"
    echo ""
    echo "🎉 现在可以启动前端服务测试ChatFlow功能了！"
else
    echo "❌ 同步失败，请检查错误信息"
    exit 1
fi