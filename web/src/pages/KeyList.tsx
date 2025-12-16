import React, { useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, message, Popconfirm, Select, Radio, Card, Tag, Space, Typography, Tooltip } from 'antd';
import { PlusOutlined, DeleteOutlined, KeyOutlined, SafetyCertificateOutlined, EditOutlined } from '@ant-design/icons';
import axios from 'axios';
import { useTranslation } from 'react-i18next';

const { Text, Paragraph } = Typography;

interface ApiKey {
  id: number;
  key: string;
  description: string;
  allowed_servers: string; // JSON string
  allowed_tools: string;   // JSON string
}

interface Server {
  id: number;
  name: string;
}

interface Tool {
    name: string;
}

const KeyList: React.FC = () => {
  const { t } = useTranslation();
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [tools, setTools] = useState<Tool[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [editingId, setEditingId] = useState<number | null>(null);
  
  // Permission Mode: 'server' or 'tool'
  const [permMode, setPermMode] = useState<'server' | 'tool'>('server');

  const fetchData = async () => {
    try {
      const [kRes, sRes, tRes] = await Promise.all([
        axios.get('/api/v1/keys'),
        axios.get('/api/v1/servers'),
        axios.get('/api/v1/tools')
      ]);
      setKeys(kRes.data);
      setServers(sRes.data);
      setTools(tRes.data || []);
    } catch (err) {
      message.error(t('common.error'));
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      
      // Clean up based on mode
      if (permMode === 'server') {
          values.allowed_tools = "";
          if (values.allowed_servers) {
            values.allowed_servers = JSON.stringify(values.allowed_servers);
          }
      } else {
          values.allowed_servers = "";
          if (values.allowed_tools) {
            values.allowed_tools = JSON.stringify(values.allowed_tools);
          }
      }
      
      if (editingId) {
          await axios.put(`/api/v1/keys/${editingId}`, values);
          message.success(t('common.success'));
      } else {
          await axios.post('/api/v1/keys', values);
          message.success(t('common.success'));
      }
      
      setIsModalOpen(false);
      form.resetFields();
      setEditingId(null);
      fetchData();
    } catch (err) {
      message.error(t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    await axios.delete(`/api/v1/keys/${id}`);
    message.success(t('common.success'));
    fetchData();
  };

  const columns = [
    { 
        title: t('key.key'), 
        dataIndex: 'key', 
        key: 'key',
        width: 300,
        render: (key: string) => (
            <div style={{ display: 'flex', alignItems: 'center', background: '#f5f7fa', padding: '4px 8px', borderRadius: 4, border: '1px solid #eee' }}>
                <KeyOutlined style={{ color: '#ccc', marginRight: 8 }} />
                <Paragraph copyable={{ text: key, tooltips: ['Copy', 'Copied!'] }} style={{ marginBottom: 0, fontFamily: 'monospace', color: '#666' }}>
                    {key.substring(0, 12)}...{key.substring(key.length - 4)}
                </Paragraph>
            </div>
        )
    },
    { 
        title: t('key.description'), 
        dataIndex: 'description', 
        key: 'description',
        render: (text: string) => <Text strong>{text}</Text>
    },
    { 
      title: t('key.permissions'), 
      key: 'permissions',
      render: (_: any, record: ApiKey) => {
        // Check Tool Permissions first
        if (record.allowed_tools && record.allowed_tools !== "") {
             try {
                const toolNames = JSON.parse(record.allowed_tools);
                if (toolNames.length === 0) return <Tag color="default">No Access</Tag>;
                if (toolNames.includes('*')) return <Tag color="gold" icon={<SafetyCertificateOutlined />}>All Tools</Tag>;
                return <Tag color="blue">{toolNames.length} Tools Allowed</Tag>;
             } catch { return <Tag color="error">Invalid Config</Tag>; }
        }
        
        // Fallback to Server Permissions
        try {
          const ids = JSON.parse(record.allowed_servers || '[]');
          if (ids.length === 0) return <Tag color="gold" icon={<SafetyCertificateOutlined />}>All Servers</Tag>;
          return (
            <Space size={[0, 4]} wrap>
                {ids.map((id: string) => {
                    const s = servers.find(s => String(s.id) === id);
                    return <Tag key={id}>{s ? s.name : id}</Tag>;
                })}
            </Space>
          );
        } catch {
          return record.allowed_servers;
        }
      }
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 120,
      render: (_: any, record: ApiKey) => (
        <Space>
            <Tooltip title={t('key.edit_key')}>
                <Button type="text" icon={<EditOutlined />} onClick={() => {
                    setEditingId(record.id);
                    
                    // Parse existing permissions to populate form
                    let mode: 'server' | 'tool' = 'server';
                    let serversVal: string[] = [];
                    let toolsVal: string[] = [];
                    
                    if (record.allowed_tools && record.allowed_tools !== "") {
                        mode = 'tool';
                        try {
                            toolsVal = JSON.parse(record.allowed_tools);
                        } catch {}
                    } else {
                        try {
                            serversVal = JSON.parse(record.allowed_servers || '[]');
                        } catch {}
                    }
                    
                    setPermMode(mode);
                    form.setFieldsValue({
                        description: record.description,
                        allowed_servers: serversVal,
                        allowed_tools: toolsVal
                    });
                    
                    setIsModalOpen(true);
                }} />
            </Tooltip>
            <Popconfirm title={t('common.confirm_delete')} onConfirm={() => handleDelete(record.id)} okText={t('common.delete')} okButtonProps={{ danger: true }} cancelText={t('common.cancel')}>
                <Button type="text" danger icon={<DeleteOutlined />} />
            </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card className="premium-card" bodyStyle={{ padding: '0' }} bordered={false}>
      <div style={{ padding: '16px 24px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: 16, fontWeight: 500 }}>{t('key.title')}</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
            setEditingId(null);
            setIsModalOpen(true);
            setPermMode('server'); // Default
            form.resetFields();
        }}>{t('key.add_key')}</Button>
      </div>
      
      <Table dataSource={keys} columns={columns} rowKey="id" pagination={{ pageSize: 10 }} />
      
      <Modal title={editingId ? t('key.edit_key') : t('key.add_key')} open={isModalOpen} onOk={handleOk} onCancel={() => setIsModalOpen(false)}>
        <Form form={form} layout="vertical" style={{ marginTop: 24 }}>
          <Form.Item name="description" label={t('key.description')} rules={[{ required: true }]} tooltip="Friendly name to identify who uses this key">
            <Input size="large" placeholder="e.g. Cursor Client" />
          </Form.Item>
          
          <Form.Item label={t('key.permission_mode')} style={{ marginBottom: 12 }}>
            <Radio.Group 
                value={permMode} 
                onChange={e => setPermMode(e.target.value)} 
                optionType="button"
                buttonStyle="solid"
            >
                <Radio.Button value="server">{t('key.mode_server')}</Radio.Button>
                <Radio.Button value="tool">{t('key.mode_tool')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <div style={{ background: '#fafafa', padding: 16, borderRadius: 8, border: '1px solid #eee' }}>
            {permMode === 'server' && (
                <Form.Item name="allowed_servers" label={t('key.allowed_servers')} style={{ marginBottom: 0 }}>
                    <Select mode="multiple" placeholder={t('key.select_servers')} size="large">
                    {servers.map(s => (
                        <Select.Option key={s.id} value={String(s.id)}>{s.name}</Select.Option>
                    ))}
                    </Select>
                </Form.Item>
            )}

            {permMode === 'tool' && (
                <Form.Item name="allowed_tools" label={t('key.allowed_tools')} style={{ marginBottom: 0 }}>
                    <Select mode="multiple" placeholder={t('key.select_tools')} size="large">
                    <Select.Option key="*" value="*">All Tools (*)</Select.Option>
                    {tools.map(t => (
                        <Select.Option key={t.name} value={t.name}>{t.name}</Select.Option>
                    ))}
                    </Select>
                </Form.Item>
            )}
          </div>
        </Form>
      </Modal>
    </Card>
  );
};

export default KeyList;
