import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useTheme } from '../theme/theme'
import { Button } from './ui'

const nav = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/servers', label: 'Servers' },
  { to: '/clients', label: 'Clients' },
  { to: '/settings', label: 'Settings' },
  { to: '/logs', label: 'Logs' },
]

export function Layout() {
  const { admin, logout } = useAuth()
  const { theme, toggle } = useTheme()
  const navigate = useNavigate()

  const onLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-neutral-50 text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100">
      <div className="mx-auto flex max-w-6xl gap-6 px-4 py-6">
        <aside className="w-52 shrink-0">
          <div className="mb-6 px-2">
            <h1 className="text-lg font-semibold">Xray Panel</h1>
            <p className="text-xs text-neutral-500">{admin?.username}</p>
          </div>
          <nav className="space-y-1">
            {nav.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `block rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-indigo-600 text-white'
                      : 'text-neutral-600 hover:bg-neutral-200 dark:text-neutral-300 dark:hover:bg-neutral-800'
                  }`
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
          <div className="mt-6 space-y-2 px-1">
            <Button variant="ghost" className="w-full justify-start" onClick={toggle}>
              {theme === 'dark' ? '☀ Light' : '☾ Dark'}
            </Button>
            <Button variant="ghost" className="w-full justify-start" onClick={onLogout}>
              ⏻ Logout
            </Button>
          </div>
        </aside>
        <main className="min-w-0 flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
