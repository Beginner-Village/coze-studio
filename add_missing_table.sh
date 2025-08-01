#!/bin/bash

# 添加缺失的数据库表脚本

echo "=== 添加缺失的ChatFlow数据库表 ==="
echo "目标数据库: 10.10.10.224:3306"
echo ""

# 数据库配置
MYSQL_HOST="10.10.10.224"
MYSQL_PORT="3306"
MYSQL_USER="coze"
MYSQL_PASSWORD="coze123"
MYSQL_DATABASE="opencoze"

# 检查SQL文件是否存在
if [ ! -f "add_missing_table.sql" ]; then
    echo "❌ 错误: 未找到 add_missing_table.sql 文件"
    exit 1
fi

echo "🔍 检查当前表是否已存在..."

# 检查表是否已存在
TABLE_EXISTS=$(mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "USE $MYSQL_DATABASE; SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$MYSQL_DATABASE' AND table_name = 'chatflow_conversation_history';" -s -N 2>/dev/null)

if [ "$TABLE_EXISTS" = "1" ]; then
    echo "✅ chatflow_conversation_history 表已存在"
    echo "🔍 检查表结构..."
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "USE $MYSQL_DATABASE; DESCRIBE chatflow_conversation_history;" 2>/dev/null
    echo ""
    echo "表已存在，无需创建。如需重建请先手动删除表。"
    exit 0
fi

echo "📝 开始添加缺失的表..."

# 执行SQL脚本
if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" < add_missing_table.sql; then
    echo ""
    echo "✅ 缺失的表添加完成！"
    echo ""
    echo "已添加的表:"
    echo "  - chatflow_conversation_history (ChatFlow对话历史表)"
    echo ""
    echo "🔍 验证表结构:"
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "USE $MYSQL_DATABASE; DESCRIBE chatflow_conversation_history;" 2>/dev/null
    echo ""
    echo "🎉 数据库表结构现在完整了！"
else
    echo "❌ 添加表失败，请检查错误信息"
    exit 1
fi