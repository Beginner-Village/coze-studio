/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState, useEffect } from 'react';
import {
  Button,
  Avatar,
  IconButton,
  Dropdown,
  Input,
  Modal,
  Spin,
  Empty,
  Toast,
  Table,
} from '@coze-arch/coze-design';
import {
  IconCozPlus,
  IconCozMore,
} from '@coze-arch/coze-design/icons';
import { admin } from '@coze-studio/api-schema';

interface AdminUser {
  id: string;
  user_id: string;
  user_name?: string;
  avatar_url?: string;
  role: string;
  created_at: number;
  created_by: string;
}

export const AdminUsersPage: React.FC = () => {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [newUserId, setNewUserId] = useState('');
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const response = await admin.ListAdminUsers({});
      if (response.code === 0 && response.items) {
        setUsers(response.items as AdminUser[]);
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error;
        if (responseData.items) {
          setUsers(responseData.items as AdminUser[]);
        }
      } else {
        console.error('获取管理员列表失败:', error);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleAddAdmin = async () => {
    if (!newUserId) {
      Toast.error('请输入用户ID');
      return;
    }

    setAdding(true);
    try {
      const response = await admin.AddAdmin({ user_id: newUserId, role: 'admin' });
      if (response.code === 0) {
        Toast.success('添加成功');
        setShowAddModal(false);
        setNewUserId('');
        fetchUsers();
      } else {
        Toast.error(response.msg || '添加失败');
      }
    } catch (error: any) {
      if (error.code === '0' || error.code === 0) {
        Toast.success('添加成功');
        setShowAddModal(false);
        setNewUserId('');
        fetchUsers();
      } else {
        console.error('添加失败:', error);
        Toast.error('添加失败');
      }
    } finally {
      setAdding(false);
    }
  };

  const handleRemoveAdmin = async (userId: string) => {
    if (!confirm('确定要移除该管理员吗？')) {
      return;
    }

    try {
      const response = await admin.RemoveAdmin({ user_id: userId });
      if (response.code === 0) {
        Toast.success('移除成功');
        fetchUsers();
      } else {
        Toast.error(response.msg || '移除失败');
      }
    } catch (error: any) {
      if (error.code === '0' || error.code === 0) {
        Toast.success('移除成功');
        fetchUsers();
      } else {
        console.error('移除失败:', error);
        Toast.error('移除失败');
      }
    }
  };

  const columns = [
    {
      title: '用户',
      dataIndex: 'user_id',
      key: 'user_id',
      render: (value: string, record: AdminUser) => (
        <div className="flex items-center gap-3">
          <Avatar size="small">
            {record.avatar_url ? (
              <img src={record.avatar_url} alt="" className="w-full h-full object-cover" />
            ) : (
              <span>{record.user_name?.[0] || 'U'}</span>
            )}
          </Avatar>
          <div>
            <div className="font-medium">{record.user_name || `用户 ${value}`}</div>
            <div className="text-xs text-gray-500">ID: {value}</div>
          </div>
        </div>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (value: string) => (
        <span
          className={`px-2 py-1 rounded text-xs font-medium ${
            value === 'super_admin'
              ? 'bg-purple-100 text-purple-700'
              : 'bg-blue-100 text-blue-700'
          }`}
        >
          {value === 'super_admin' ? '超级管理员' : '管理员'}
        </span>
      ),
    },
    {
      title: '添加时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value: number) => {
        if (!value) return '-';
        const date = new Date(value * 1000);
        return date.toLocaleString('zh-CN');
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: any, record: AdminUser) => {
        // 超级管理员不可移除
        if (record.role === 'super_admin') {
          return <span className="text-gray-400 text-xs">-</span>;
        }
        return (
          <Dropdown
            trigger="click"
            position="bottomRight"
            render={
              <Dropdown.Menu>
                <Dropdown.Item
                  type="danger"
                  onClick={() => handleRemoveAdmin(record.user_id)}
                >
                  移除
                </Dropdown.Item>
              </Dropdown.Menu>
            }
          >
            <IconButton icon={<IconCozMore />} />
          </Dropdown>
        );
      },
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      {/* 头部操作栏 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <p className="text-sm text-gray-500">
            管理后台访问权限控制。超级管理员可以管理公共模型和其他管理员。
          </p>
        </div>

        <Button
          type="primary"
          icon={<IconCozPlus />}
          onClick={() => setShowAddModal(true)}
        >
          添加管理员
        </Button>
      </div>

      {/* 管理员列表 */}
      {users.length === 0 ? (
        <Empty description="暂无管理员" />
      ) : (
        <div className="bg-white rounded-lg border">
          <Table
            columns={columns}
            dataSource={users}
            rowKey="id"
            pagination={false}
          />
        </div>
      )}

      {/* 添加管理员弹窗 */}
      <Modal
        title="添加管理员"
        visible={showAddModal}
        onCancel={() => {
          setShowAddModal(false);
          setNewUserId('');
        }}
        footer={
          <div className="flex justify-end gap-3">
            <Button onClick={() => setShowAddModal(false)}>取消</Button>
            <Button type="primary" loading={adding} onClick={handleAddAdmin}>
              确认添加
            </Button>
          </div>
        }
      >
        <div className="py-4">
          <label className="block text-sm font-medium mb-2">用户 ID</label>
          <Input
            value={newUserId}
            onChange={(val) => setNewUserId(val)}
            placeholder="请输入要添加的用户 ID"
          />
          <p className="text-xs text-gray-400 mt-2">
            输入用户 ID 后，该用户将获得管理后台的访问权限
          </p>
        </div>
      </Modal>
    </div>
  );
};

export { AdminUsersPage as Component };
export default AdminUsersPage;
