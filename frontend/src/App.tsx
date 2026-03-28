import { BrowserRouter, Routes, Route, useLocation, useNavigate } from 'react-router-dom'
import { lazy, Suspense, useState, useEffect } from 'react'
import { ConfigProvider, Layout, Menu, Button, Spin } from 'antd'
import {
  DashboardOutlined,
  UserOutlined,
  GlobalOutlined,
  HistoryOutlined,
  SettingOutlined,
  SunOutlined,
  MoonOutlined,
} from '@ant-design/icons'
import zhCN from 'antd/locale/zh_CN'
import { apiFetch } from '@/lib/utils'
import type { PlatformMeta } from '@/lib/registerOptions'
import { darkTheme, lightTheme } from './theme'

const { Sider, Content } = Layout
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Accounts = lazy(() => import('@/pages/Accounts'))
const Register = lazy(() => import('@/pages/Register'))
const Proxies = lazy(() => import('@/pages/Proxies'))
const Settings = lazy(() => import('@/pages/Settings'))
const TaskHistory = lazy(() => import('@/pages/TaskHistory'))

function AppContent() {
  const [themeMode, setThemeMode] = useState<'dark' | 'light'>(() =>
    (localStorage.getItem('theme') as 'dark' | 'light') || 'dark'
  )
  const [collapsed, setCollapsed] = useState(false)
  const [platforms, setPlatforms] = useState<PlatformMeta[]>([])
  const location = useLocation()
  const navigate = useNavigate()

  useEffect(() => {
    document.documentElement.classList.toggle('light', themeMode === 'light')
    document.documentElement.style.setProperty(
      '--sider-trigger-border',
      themeMode === 'light' ? 'rgba(0,0,0,0.1)' : 'rgba(255,255,255,0.15)'
    )
    localStorage.setItem('theme', themeMode)
  }, [themeMode])

  useEffect(() => {
    apiFetch('/platforms')
      .then((items) => setPlatforms(items || []))
      .catch(() => setPlatforms([]))
  }, [])

  const isLight = themeMode === 'light'
  const currentTheme = isLight ? lightTheme : darkTheme

  const getSelectedKey = () => {
    const path = location.pathname
    if (path === '/') return ['/']
    if (path.startsWith('/accounts')) return [path]
    if (path === '/history') return ['/history']
    if (path === '/proxies') return ['/proxies']
    if (path === '/settings') return ['/settings']
    return ['/']
  }

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: '仪表盘',
    },
    {
      key: '/accounts',
      icon: <UserOutlined />,
      label: '平台管理',
      children: platforms.map(p => ({
        key: `/accounts/${p.name}`,
        label: p.available === false ? `${p.display_name} (不可用)` : p.display_name,
      })),
    },
    {
      key: '/history',
      icon: <HistoryOutlined />,
      label: '任务历史',
    },
    {
      key: '/proxies',
      icon: <GlobalOutlined />,
      label: '代理管理',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: '全局配置',
    },
  ]

  return (
    <ConfigProvider theme={currentTheme} locale={zhCN}>
      <Layout style={{ minHeight: '100vh' }}>
        <Sider
          collapsible
          collapsed={collapsed}
          onCollapse={setCollapsed}
          style={{
            background: currentTheme.token?.colorBgContainer,
            borderRight: `1px solid ${currentTheme.token?.colorBorder}`,
          }}
          width={220}
        >
          <div
            style={{
              height: 64,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderBottom: `1px solid ${currentTheme.token?.colorBorder}`,
            }}
          >
            <DashboardOutlined style={{ fontSize: 20, color: currentTheme.token?.colorPrimary }} />
            {!collapsed && (
              <span
                style={{
                  marginLeft: 8,
                  fontWeight: 600,
                  fontSize: 14,
                  color: currentTheme.token?.colorText,
                }}
              >
                Account Manager
              </span>
            )}
          </div>
          <Menu
            mode="inline"
            selectedKeys={getSelectedKey()}
            defaultOpenKeys={['/accounts']}
            items={menuItems}
            onClick={({ key }) => navigate(key)}
            style={{
              borderRight: 0,
              background: 'transparent',
            }}
          />
          <div
            style={{
              position: 'absolute',
              bottom: 16,
              left: 0,
              right: 0,
              padding: '0 16px',
            }}
          >
            <Button
              block
              icon={isLight ? <SunOutlined /> : <MoonOutlined />}
              onClick={() => setThemeMode(isLight ? 'dark' : 'light')}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: collapsed ? 'center' : 'space-between',
              }}
            >
              {!collapsed && (isLight ? '亮色模式' : '暗色模式')}
            </Button>
          </div>
        </Sider>
        <Content
          style={{
            padding: 24,
            overflow: 'auto',
            background: currentTheme.token?.colorBgLayout,
          }}
        >
          <Suspense
            fallback={
              <div style={{ minHeight: 320, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Spin size="large" />
              </div>
            }
          >
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/accounts" element={<Accounts />} />
              <Route path="/accounts/:platform" element={<Accounts />} />
              <Route path="/register" element={<Register />} />
              <Route path="/history" element={<TaskHistory />} />
              <Route path="/proxies" element={<Proxies />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </Suspense>
        </Content>
      </Layout>
    </ConfigProvider>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  )
}
