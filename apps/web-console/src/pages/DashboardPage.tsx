import { Link } from 'react-router-dom'

export function DashboardPage() {
  return (
    <section>
      <div className="page-heading">
        <div>
          <p className="eyebrow">Environment overview</p>
          <h1>Ephemeral environments</h1>
          <p>Provisioning state and controls will appear here.</p>
        </div>
        <Link className="button-link" to="/environments/new">
          Create environment
        </Link>
      </div>
      <div className="placeholder-card">
        <h2>No environments loaded yet</h2>
        <p>The dashboard query and environment table will be added in the next UI milestone.</p>
      </div>
    </section>
  )
}
