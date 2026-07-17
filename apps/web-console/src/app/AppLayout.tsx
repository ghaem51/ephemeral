import { NavLink, Outlet } from 'react-router-dom'

export function AppLayout() {
  return (
    <div className="app-shell">
      <header className="app-header">
        <NavLink className="brand" to="/">
          <span className="brand-mark">EP</span>
          <span>
            <strong>EnvPilot</strong>
            <small>Ephemeral environments</small>
          </span>
        </NavLink>
        <nav aria-label="Primary navigation">
          <NavLink to="/" end>
            Environments
          </NavLink>
          <NavLink to="/environments/new">Create environment</NavLink>
        </nav>
      </header>
      <main className="page-shell">
        <Outlet />
      </main>
    </div>
  )
}
