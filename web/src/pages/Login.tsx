import React, { useState } from 'react';
import { Form, Input, Button, message, Typography, theme } from 'antd';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import { UserOutlined, LockOutlined, ApiOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const { token } = theme.useToken();

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const res = await axios.post('/api/login', values);
      localStorage.setItem('token', res.data.token);
      message.success(t('common.success'));
      navigate('/');
    } catch (err) {
      message.error(t('login.failed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      {/* Left Side - Artistic Background */}
      <div className="login-bg" style={{ 
        flex: 1, 
        display: 'flex', 
        flexDirection: 'column', 
        justifyContent: 'center', 
        alignItems: 'center',
        color: 'white',
        padding: 40,
        position: 'relative'
      }}>
        <div style={{ 
          background: 'rgba(255, 255, 255, 0.1)', 
          backdropFilter: 'blur(10px)', 
          padding: 40, 
          borderRadius: 16,
          textAlign: 'center',
          maxWidth: 500
        }}>
          <ApiOutlined style={{ fontSize: 64, marginBottom: 24 }} />
          <Title level={1} style={{ color: 'white', margin: 0 }}>{t('login.title')}</Title>
          <Text style={{ color: 'rgba(255,255,255,0.85)', fontSize: 18, marginTop: 16, display: 'block' }}>
            {t('login.subtitle')}
          </Text>
        </div>
      </div>

      {/* Right Side - Login Form */}
      <div style={{ 
        flex: '0 0 500px', 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        background: 'white',
        boxShadow: '-4px 0 16px rgba(0,0,0,0.05)',
        zIndex: 1
      }}>
        <div style={{ width: '100%', maxWidth: 360, padding: 24 }}>
          <div style={{ textAlign: 'center', marginBottom: 40 }}>
            <Title level={2} style={{ marginBottom: 8 }}>{t('login.signin')}</Title>
          </div>

          <Form
            name="login"
            onFinish={onFinish}
            layout="vertical"
            size="large"
          >
            <Form.Item
              name="username"
              rules={[{ required: true }]}
            >
              <Input 
                prefix={<UserOutlined style={{ color: token.colorTextQuaternary }} />} 
                placeholder={t('common.username')}
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[{ required: true }]}
            >
              <Input.Password 
                prefix={<LockOutlined style={{ color: token.colorTextQuaternary }} />} 
                placeholder={t('common.password')}
              />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" block loading={loading} style={{ height: 48, fontSize: 16 }}>
                {t('common.login')}
              </Button>
            </Form.Item>
            
            <div style={{ textAlign: 'center', marginTop: 16 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>Default: admin / admin</Text>
            </div>
          </Form>
        </div>
      </div>
    </div>
  );
};

export default Login;
