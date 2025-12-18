import { Outlet, Link, useLocation } from 'react-router-dom';
import { Layout, Menu, Typography } from 'antd';
import { DashboardOutlined } from '@ant-design/icons';

const { Header, Content, Footer } = Layout;
const { Title } = Typography;

export function App() {
  const location = useLocation();

  const menuItems = [
    {
      key: '/admin',
      icon: <DashboardOutlined />,
      label: <Link to="/admin">Dashboard</Link>,
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <Title level={4} style={{ color: 'white', margin: 0 }}>
          AI Gateway Admin
        </Title>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[location.pathname]}
          items={menuItems}
          style={{ flex: 1 }}
        />
      </Header>
      <Content style={{ background: '#f0f2f5' }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center' }}>
        AI Gateway Admin ©{new Date().getFullYear()}
      </Footer>
    </Layout>
  );
}
