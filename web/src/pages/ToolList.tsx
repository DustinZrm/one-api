import React, { useEffect, useState } from 'react';
import { Table, Button, message, Tag, Card, Input, Space, Typography } from 'antd';
import { ReloadOutlined, SearchOutlined, CodeOutlined } from '@ant-design/icons';
import axios from 'axios';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

interface Tool {
  name: string;
  description?: string;
  inputSchema?: any;
}

const ToolList: React.FC = () => {
  const { t } = useTranslation();
  const [tools, setTools] = useState<Tool[]>([]);
  const [filteredTools, setFilteredTools] = useState<Tool[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState('');

  const fetchTools = async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/v1/tools');
      const data = res.data || [];
      setTools(data);
      setFilteredTools(data);
    } catch (err) {
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTools();
  }, []);

  useEffect(() => {
      if (!searchText) {
          setFilteredTools(tools);
      } else {
          const lower = searchText.toLowerCase();
          setFilteredTools(tools.filter(t => 
              t.name.toLowerCase().includes(lower) || 
              (t.description && t.description.toLowerCase().includes(lower))
          ));
      }
  }, [searchText, tools]);

  const columns = [
    { 
      title: t('server.tool_name'), 
      dataIndex: 'name', 
      key: 'name',
      render: (name: string) => {
        const parts = name.split('__');
        if (parts.length === 2) {
            return (
                <Space>
                    <Tag color="geekblue" style={{ marginRight: 0 }}>{parts[0]}</Tag>
                    <span style={{ color: '#ccc' }}>/</span>
                    <Text strong>{parts[1]}</Text>
                </Space>
            );
        }
        return <Text strong>{name}</Text>;
      }
    },
    { 
      title: t('server.tool_description'), 
      dataIndex: 'description', 
      key: 'description',
      ellipsis: true,
      render: (text: string) => <Text type="secondary">{text || 'No description provided'}</Text>
    },
  ];

  return (
    <Card className="premium-card" bodyStyle={{ padding: '0' }} bordered={false}>
      <div style={{ padding: '16px 24px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
            <span style={{ fontSize: 16, fontWeight: 500 }}>{t('tool.title')}</span>
            <Tag color="blue">{filteredTools.length}</Tag>
        </Space>
        <Space>
            <Input 
                placeholder={t('common.search')} 
                prefix={<SearchOutlined style={{ color: '#ccc' }} />} 
                value={searchText}
                onChange={e => setSearchText(e.target.value)}
                style={{ width: 250 }}
                allowClear
            />
            <Button icon={<ReloadOutlined />} onClick={fetchTools} loading={loading}>{t('common.refresh')}</Button>
        </Space>
      </div>

      <Table 
        dataSource={filteredTools} 
        columns={columns} 
        rowKey="name" 
        loading={loading}
        pagination={{ pageSize: 10 }}
        expandable={{
            expandedRowRender: (record) => (
                <div style={{ padding: '16px', background: '#fafafa', borderRadius: 8 }}>
                    <div style={{ marginBottom: 8 }}>
                        <CodeOutlined /> <Text strong>{t('tool.schema_title')}</Text>
                    </div>
                    <pre style={{ 
                        margin: 0, 
                        background: '#fff', 
                        padding: 12, 
                        borderRadius: 6, 
                        border: '1px solid #eee',
                        fontSize: 12,
                        color: '#666',
                        overflowX: 'auto'
                    }}>
                        {JSON.stringify(record.inputSchema, null, 2)}
                    </pre>
                </div>
            ),
            rowExpandable: (record) => !!record.inputSchema
        }}
      />
    </Card>
  );
};

export default ToolList;
