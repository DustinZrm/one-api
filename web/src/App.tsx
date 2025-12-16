import React, { useState } from 'react';
import { Layout, Menu, theme, Button, Avatar, Dropdown, Space, Typography, Breadcrumb, Modal, Form, Input, message } from 'antd';
import { DesktopOutlined, KeyOutlined, ToolOutlined, LogoutOutlined, UserOutlined, ApiOutlined, MenuUnfoldOutlined, MenuFoldOutlined, LockOutlined, GlobalOutlined } from '@ant-design/icons';
import { BrowserRouter, Routes, Route, Link, useLocation, useNavigate, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import ServerList from './pages/ServerList';
import KeyList from './pages/KeyList';
import ToolList from './pages/ToolList';
import Login from './pages/Login';
import axios from 'axios';

const { Header, Content, Sider } = Layout;
const { Text } = Typography;

// Axios Interceptor Setup
axios.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response && error.response.status === 401) {
            localStorage.removeItem('token');
            if (!window.location.pathname.includes('/login')) {
                // We can't use t() here easily outside component, but that's okay
                // message.error('Session expired. Please login again.'); 
                window.location.href = '/login';
            }
        }
        return Promise.reject(error);
    }
);

// Auth Guard
const RequireAuth = ({ children }: { children: JSX.Element }) => {
    const token = localStorage.getItem('token');
    if (!token) {
        return <Navigate to="/login" replace />;
    }
    axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
    return children;
};

const MainLayout: React.FC = () => {
  const { t, i18n } = useTranslation();
  const {
    token: { colorBgContainer, colorBgLayout },
  } = theme.useToken();
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  // Change Password State
  const [isPasswordModalOpen, setIsPasswordModalOpen] = useState(false);
  const [passwordForm] = Form.useForm();

  const handleLogout = () => {
      localStorage.removeItem('token');
      navigate('/login');
  };

  const handleChangePassword = async () => {
      try {
          const values = await passwordForm.validateFields();
          if (values.new_password !== values.confirm_password) {
              message.error(t('common.password_mismatch'));
              return;
          }
          
          await axios.post('/api/v1/change-password', {
              old_password: values.old_password,
              new_password: values.new_password
          });
          
          message.success(t('common.password_changed'));
          setIsPasswordModalOpen(false);
          passwordForm.resetFields();
      } catch (err: any) {
          if (err.response?.data?.error) {
              message.error(err.response.data.error);
          } else {
              message.error(t('common.error'));
          }
      }
  };

  const userMenu = (
      <Menu items={[
          { key: 'password', icon: <LockOutlined />, label: t('common.change_password'), onClick: () => setIsPasswordModalOpen(true) },
          { key: 'logout', icon: <LogoutOutlined />, label: t('common.logout'), onClick: handleLogout }
      ]} />
  );

  const langMenu = (
      <Menu 
        selectedKeys={[i18n.language]}
        items={[
          { key: 'en', label: 'English', onClick: () => i18n.changeLanguage('en') },
          { key: 'zh', label: '简体中文', onClick: () => i18n.changeLanguage('zh') }
      ]} />
  );

  const getPageTitle = () => {
      switch(location.pathname) {
          case '/': return t('menu.servers');
          case '/tools': return t('menu.tools');
          case '/keys': return t('menu.keys');
          default: return t('menu.dashboard');
      }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider 
        trigger={null} 
        collapsible 
        collapsed={collapsed}
        width={240}
        style={{
            boxShadow: '2px 0 8px 0 rgba(29,35,41,.05)',
            zIndex: 10
        }}
      >
        <div style={{ 
            height: 64, 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            borderBottom: '1px solid rgba(255,255,255,0.1)'
        }}>
             <Space size={12}>
                <div style={{ 
                    width: 32, 
                    height: 32, 
                    background: '#1677ff', 
                    borderRadius: 6, 
                    display: 'flex', 
                    alignItems: 'center', 
                    justifyContent: 'center' 
                }}>
                    <ApiOutlined style={{ color: 'white', fontSize: 20 }} />
                </div>
                {!collapsed && (
                    <Text strong style={{ color: 'white', fontSize: 18, whiteSpace: 'nowrap' }}>
                        One MCP
                    </Text>
                )}
             </Space>
        </div>

        <Menu 
            theme="dark" 
            selectedKeys={[location.pathname]} 
            mode="inline"
            items={[
                { key: '/', icon: <DesktopOutlined />, label: <Link to="/">{t('menu.servers')}</Link> },
                { key: '/tools', icon: <ToolOutlined />, label: <Link to="/tools">{t('menu.tools')}</Link> },
                { key: '/keys', icon: <KeyOutlined />, label: <Link to="/keys">{t('menu.keys')}</Link> }
            ]}
            style={{ marginTop: 16 }}
        />
      </Sider>
      
      <Layout style={{ background: colorBgLayout }}>
        <Header style={{ 
            padding: '0 24px', 
            background: colorBgContainer, 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center',
            boxShadow: '0 1px 4px rgba(0,21,41,.08)',
            zIndex: 9
        }}>
            <Space>
                <Button
                    type="text"
                    icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                    onClick={() => setCollapsed(!collapsed)}
                    style={{
                        fontSize: '16px',
                        width: 64,
                        height: 64,
                    }}
                />
                <Breadcrumb items={[{ title: t('menu.home') }, { title: getPageTitle() }]} />
            </Space>

            <Space size={16}>
                <Dropdown overlay={langMenu} placement="bottomRight">
                    <Button type="text" icon={<GlobalOutlined />}>
                        {i18n.language === 'zh' ? '简体中文' : 'English'}
                    </Button>
                </Dropdown>

                <Dropdown overlay={userMenu} placement="bottomRight">
                    <Space style={{ cursor: 'pointer', padding: '4px 8px', borderRadius: 6 }} className="user-dropdown">
                        <Avatar icon={<UserOutlined />} style={{ backgroundColor: '#1677ff' }} />
                        <Text strong>Admin</Text>
                    </Space>
                </Dropdown>
            </Space>
        </Header>

        <Content style={{ margin: '24px', minHeight: 280 }}>
            <div style={{ marginBottom: 24 }}>
                <Text style={{ fontSize: 24, fontWeight: 600 }}>{getPageTitle()}</Text>
            </div>
            <div
                style={{
                // padding: 24,
                // background: colorBgContainer,
                // borderRadius: borderRadiusLG,
                }}
            >
                <Routes>
                    <Route path="/" element={<ServerList />} />
                    <Route path="/tools" element={<ToolList />} />
                    <Route path="/keys" element={<KeyList />} />
                </Routes>
            </div>
        </Content>
      </Layout>

      <Modal
        title={t('common.change_password')}
        open={isPasswordModalOpen}
        onOk={handleChangePassword}
        onCancel={() => {
            setIsPasswordModalOpen(false);
            passwordForm.resetFields();
        }}
      >
          <Form form={passwordForm} layout="vertical" style={{ marginTop: 24 }}>
              <Form.Item 
                name="old_password" 
                label={t('common.old_password')}
                rules={[{ required: true, message: 'Please input your current password!' }]}
              >
                  <Input.Password />
              </Form.Item>
              <Form.Item 
                name="new_password" 
                label={t('common.new_password')}
                rules={[{ required: true, message: 'Please input your new password!' }]}
              >
                  <Input.Password />
              </Form.Item>
              <Form.Item 
                name="confirm_password" 
                label={t('common.confirm_password')}
                rules={[
                    { required: true, message: 'Please confirm your new password!' },
                    ({ getFieldValue }) => ({
                        validator(_, value) {
                            if (!value || getFieldValue('new_password') === value) {
                                return Promise.resolve();
                            }
                            return Promise.reject(new Error(t('common.password_mismatch')));
                        },
                    }),
                ]}
              >
                  <Input.Password />
              </Form.Item>
          </Form>
      </Modal>
    </Layout>
  );
};

const App: React.FC = () => {
  return (
    <BrowserRouter>
        <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/*" element={
                <RequireAuth>
                    <MainLayout />
                </RequireAuth>
            } />
        </Routes>
    </BrowserRouter>
  );
};

export default App;
