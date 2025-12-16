import React, { useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, Switch, message, Popconfirm, Card, Tag, Space, Tooltip, Select, Row, Col, Divider } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SyncOutlined, CheckCircleOutlined, CloseCircleOutlined, CloudServerOutlined, CodeOutlined, ApiOutlined, MinusCircleOutlined } from '@ant-design/icons';
import axios from 'axios';
import { useTranslation } from 'react-i18next';

interface Server {
  id: number;
  name: string;
  transport_type: 'sse' | 'stdio' | 'http';
  url: string;
  command: string;
  args: string;
  env: string;
  tool_config: string;
  auth_token: string;
  enabled: boolean;
}

const ServerList: React.FC = () => {
  const { t } = useTranslation();
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [editingId, setEditingId] = useState<number | null>(null);
  const [transportType, setTransportType] = useState<'sse' | 'stdio' | 'http'>('sse');

  const fetchServers = async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/v1/servers');
      setServers(res.data);
    } catch (err) {
      message.error(t('common.error'));
    } finally {
        setLoading(false);
    }
  };

  useEffect(() => {
    fetchServers();
  }, []);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      
      // If HTTP, package tool config
      if (values.transport_type === 'http') {
          try {
              const headers = values.tool_headers ? JSON.parse(values.tool_headers) : {};
              const toolConfig = {
                  name: values.tool_name,
                  description: values.tool_description,
                  method: values.tool_method,
                  headers: headers,
                  parameters: values.tool_params || []
              };
              values.tool_config = JSON.stringify(toolConfig);
          } catch (e) {
              message.error("Invalid JSON in Headers");
              return;
          }
      }

      if (editingId) {
        await axios.put(`/api/v1/servers/${editingId}`, values);
        message.success(t('common.success'));
      } else {
        await axios.post('/api/v1/servers', values);
        message.success(t('common.success'));
      }
      setIsModalOpen(false);
      form.resetFields();
      setEditingId(null);
      fetchServers();
    } catch (err) {
      message.error(t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    await axios.delete(`/api/v1/servers/${id}`);
    message.success(t('common.success'));
    fetchServers();
  };

  const columns = [
    { 
        title: t('common.status'), 
        dataIndex: 'enabled', 
        key: 'enabled',
        width: 100,
        render: (v: boolean) => (
            v ? <Tag icon={<CheckCircleOutlined />} color="success">{t('common.active')}</Tag> 
              : <Tag icon={<CloseCircleOutlined />} color="error">{t('common.stopped')}</Tag>
        )
    },
    { 
        title: t('server.name'), 
        dataIndex: 'name', 
        key: 'name',
        render: (text: string) => <b style={{ fontSize: 15 }}>{text}</b>
    },
    {
        title: t('server.transport_type'),
        dataIndex: 'transport_type',
        key: 'transport_type',
        width: 120,
        render: (text: string) => {
            if (text === 'stdio') return <Tag color="blue" icon={<CodeOutlined />}>STDIO</Tag>;
            if (text === 'http') return <Tag color="orange" icon={<ApiOutlined />}>HTTP</Tag>;
            return <Tag color="green" icon={<CloudServerOutlined />}>SSE</Tag>;
        }
    },
    { 
        title: t('server.connection_details'), 
        key: 'details',
        render: (_: any, record: Server) => {
            if (record.transport_type === 'stdio') {
                return <span style={{ fontFamily: 'monospace', color: '#666' }}>{record.command} {record.args && record.args !== '[]' ? '...' : ''}</span>;
            }
            return <span style={{ color: '#666' }}>{record.url}</span>;
        }
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 150,
      render: (_: any, record: Server) => (
        <Space>
          <Tooltip title={t('common.edit')}>
            <Button type="text" icon={<EditOutlined />} onClick={() => {
                setEditingId(record.id);
                const type = record.transport_type || 'sse';
                setTransportType(type);
                
                const fields: any = { ...record, transport_type: type };
                if (record.transport_type === 'http' && record.tool_config) {
                    try {
                        const tc = JSON.parse(record.tool_config);
                        fields['tool_name'] = tc.name;
                        fields['tool_description'] = tc.description;
                        fields['tool_method'] = tc.method;
                        fields['tool_headers'] = JSON.stringify(tc.headers || {}, null, 2);
                        fields['tool_params'] = tc.parameters;
                    } catch (e) {}
                }
                
                form.setFieldsValue(fields);
                setIsModalOpen(true);
            }} />
          </Tooltip>
          <Tooltip title={t('common.delete')}>
            <Popconfirm title={t('common.confirm_delete')} onConfirm={() => handleDelete(record.id)} okText={t('common.yes')} cancelText={t('common.no')}>
                <Button type="text" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <Card className="premium-card" bodyStyle={{ padding: '0' }} bordered={false}>
      <div style={{ padding: '16px 24px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: 16, fontWeight: 500 }}>{t('server.title')}</span>
        <Space>
            <Button icon={<SyncOutlined />} onClick={fetchServers} loading={loading}>{t('common.refresh')}</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
            setEditingId(null);
            setTransportType('sse');
            form.resetFields();
            form.setFieldsValue({ transport_type: 'sse', enabled: true, tool_method: 'GET', tool_headers: '{}' });
            setIsModalOpen(true);
            }}>{t('server.add_server')}</Button>
        </Space>
      </div>
      
      <Table 
        dataSource={servers} 
        columns={columns} 
        rowKey="id" 
        pagination={{ pageSize: 10 }}
        loading={loading}
      />
      
      <Modal 
        title={editingId ? t('server.edit_server') : t('server.connect_server')} 
        open={isModalOpen} 
        onOk={handleOk} 
        onCancel={() => setIsModalOpen(false)}
        okText={editingId ? t('common.save') : t('common.submit')}
        width={800}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 24 }}>
          <Row gutter={16}>
              <Col span={16}>
                <Form.Item name="name" label={t('server.name')} rules={[{ required: true }]} tooltip={t('server.name_tooltip')}>
                    <Input size="large" placeholder="e.g. github" prefix={<span style={{ color: '#999' }}>@</span>} />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="enabled" label={t('common.status')} valuePropName="checked">
                    <Switch checkedChildren={t('common.active')} unCheckedChildren={t('common.stopped')} />
                </Form.Item>
              </Col>
          </Row>

          <Form.Item name="transport_type" label={t('server.transport_type')} rules={[{ required: true }]}>
            <Select size="large" onChange={(val) => setTransportType(val)}>
                <Select.Option value="sse">SSE (Server-Sent Events)</Select.Option>
                <Select.Option value="stdio">Stdio (Local Process)</Select.Option>
                <Select.Option value="http">HTTP / REST API (Single Tool)</Select.Option>
            </Select>
          </Form.Item>

          {transportType === 'sse' && (
            <>
                <Form.Item name="url" label={t('server.url')} rules={[{ required: true }]} tooltip={t('server.url_tooltip')}>
                    <Input size="large" placeholder="http://localhost:3000/sse" />
                </Form.Item>
                <Form.Item name="auth_token" label={t('server.auth_token')} tooltip={t('server.auth_token_tooltip')}>
                    <Input.Password size="large" placeholder="sk-..." />
                </Form.Item>
            </>
          )}

          {transportType === 'stdio' && (
            <>
                <Form.Item name="command" label={t('server.command')} rules={[{ required: true }]} tooltip={t('server.command_tooltip')}>
                    <Input size="large" placeholder="npx" />
                </Form.Item>
                <Form.Item name="args" label={t('server.args')} tooltip={t('server.args_tooltip')}>
                    <Input.TextArea autoSize={{ minRows: 2, maxRows: 6 }} placeholder='["-y", "@modelcontextprotocol/server-filesystem", "/Users/me/Documents"]' />
                </Form.Item>
                <Form.Item name="env" label={t('server.env')} tooltip={t('server.env_tooltip')}>
                    <Input.TextArea autoSize={{ minRows: 2, maxRows: 6 }} placeholder='{"GITHUB_TOKEN": "ghp_..."}' />
                </Form.Item>
            </>
          )}

          {transportType === 'http' && (
              <div style={{ background: '#fafafa', padding: 16, borderRadius: 8 }}>
                  <Form.Item name="url" label={t('server.url')} rules={[{ required: true }]}>
                    <Input size="large" placeholder="https://api.example.com/v1/weather" />
                  </Form.Item>
                  <Row gutter={16}>
                      <Col span={8}>
                        <Form.Item name="tool_method" label={t('server.tool_method')} rules={[{ required: true }]}>
                            <Select>
                                <Select.Option value="GET">GET</Select.Option>
                                <Select.Option value="POST">POST</Select.Option>
                                <Select.Option value="PUT">PUT</Select.Option>
                                <Select.Option value="DELETE">DELETE</Select.Option>
                            </Select>
                        </Form.Item>
                      </Col>
                      <Col span={16}>
                        <Form.Item name="tool_name" label={t('server.tool_name')} rules={[{ required: true }]}>
                            <Input placeholder="get_weather" />
                        </Form.Item>
                      </Col>
                  </Row>
                  
                  <Form.Item name="tool_description" label={t('server.tool_description')} rules={[{ required: true }]}>
                      <Input.TextArea placeholder="Get current weather for a city" autoSize={{ minRows: 2 }} />
                  </Form.Item>

                  <Form.Item name="tool_headers" label={t('server.tool_headers')} tooltip="Fixed headers sent with every request (e.g. API Keys)">
                      <Input.TextArea placeholder='{"Authorization": "Bearer ..."}' autoSize={{ minRows: 2 }} />
                  </Form.Item>

                  <Divider orientation="left">{t('server.parameters')}</Divider>
                  <Form.List name="tool_params">
                    {(fields, { add, remove }) => (
                        <>
                        {fields.map(({ key, name, ...restField }) => (
                            <div key={key} style={{ display: 'flex', marginBottom: 8, gap: 8, alignItems: 'flex-start' }}>
                                <Form.Item {...restField} name={[name, 'name']} rules={[{ required: true }]} style={{ width: 120, marginBottom: 0 }}>
                                    <Input placeholder={t('server.param_name')} />
                                </Form.Item>
                                <Form.Item {...restField} name={[name, 'type']} rules={[{ required: true }]} style={{ width: 100, marginBottom: 0 }}>
                                    <Select placeholder={t('server.param_type')}>
                                        <Select.Option value="string">String</Select.Option>
                                        <Select.Option value="number">Number</Select.Option>
                                        <Select.Option value="boolean">Boolean</Select.Option>
                                    </Select>
                                </Form.Item>
                                <Form.Item {...restField} name={[name, 'required']} valuePropName="checked" style={{ width: 40, marginBottom: 0 }}>
                                    <Switch size="small" />
                                </Form.Item>
                                <Form.Item {...restField} name={[name, 'default']} style={{ width: 100, marginBottom: 0 }}>
                                    <Input placeholder={t('server.param_default')} />
                                </Form.Item>
                                <Form.Item {...restField} name={[name, 'description']} style={{ flex: 1, marginBottom: 0 }}>
                                    <Input placeholder={t('server.param_desc')} />
                                </Form.Item>
                                <MinusCircleOutlined onClick={() => remove(name)} style={{ marginTop: 8 }} />
                            </div>
                        ))}
                        <Form.Item>
                            <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                                {t('server.add_param')}
                            </Button>
                        </Form.Item>
                        </>
                    )}
                  </Form.List>
              </div>
          )}
        </Form>
      </Modal>
    </Card>
  );
};

export default ServerList;
