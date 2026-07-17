import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { isActiveEnvironment, listEnvironments } from '../api/environments'
import { ApiError } from '../api/client'
import { StatusBadge } from '../components/StatusBadge'

export function DashboardPage() {
  const environments = useQuery({
    queryKey: ['environments'],
    queryFn: listEnvironments,
    refetchInterval: (query) =>
      query.state.data?.some(isActiveEnvironment) ? 1_500 : false,
  })

  return (
    <section>
      <div className="page-heading">
        <div>
          <p className="eyebrow">Environment overview</p>
          <h1>Ephemeral environments</h1>
          <p>Monitor provisioning, runtime access, and lifecycle state from one place.</p>
        </div>
        <Link className="button-link" to="/environments/new">
          Create environment
        </Link>
      </div>
      {environments.isPending ? <DashboardLoading /> : null}
      {environments.isError ? (
        <DashboardError error={environments.error} onRetry={() => void environments.refetch()} />
      ) : null}
      {environments.data?.length === 0 ? <DashboardEmpty /> : null}
      {environments.data && environments.data.length > 0 ? (
        <div className="table-card">
          <div className="table-summary">
            <strong>{environments.data.length} environments</strong>
            {environments.isFetching ? <span className="sync-indicator">Refreshing…</span> : null}
          </div>
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Status</th>
                  <th>Image</th>
                  <th>Latest operation</th>
                  <th>Created</th>
                  <th>URL</th>
                </tr>
              </thead>
              <tbody>
                {environments.data.map((environment) => (
                  <tr key={environment.id}>
                    <td>
                      <Link className="environment-link" to={`/environments/${environment.id}`}>
                        {environment.name}
                      </Link>
                    </td>
                    <td><StatusBadge status={environment.status} /></td>
                    <td><code>{environment.image}</code></td>
                    <td>{environment.latestWorkflow?.operation ?? '—'}</td>
                    <td>{formatCreatedAt(environment.createdAt)}</td>
                    <td>
                      {environment.url ? (
                        <a className="runtime-link" href={environment.url} target="_blank" rel="noreferrer">
                          Open ↗
                        </a>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function DashboardLoading() {
  return (
    <div className="state-card" role="status">
      <span className="spinner" aria-hidden="true" />
      <div><h2>Loading environments</h2><p>Connecting to the EnvPilot control plane.</p></div>
    </div>
  )
}

function DashboardEmpty() {
  return (
    <div className="state-card empty-dashboard">
      <div className="state-icon">＋</div>
      <div>
        <h2>No environments yet</h2>
        <p>Create a healthy demo or deliberately exercise the health-check failure path.</p>
        <Link className="text-link" to="/environments/new">Create your first environment →</Link>
      </div>
    </div>
  )
}

function DashboardError({ error, onRetry }: { error: Error; onRetry: () => void }) {
  const requestId = error instanceof ApiError ? error.requestId : undefined
  return (
    <div className="state-card error-card" role="alert">
      <div className="state-icon">!</div>
      <div>
        <h2>Environments could not be loaded</h2>
        <p>{error.message}</p>
        {requestId ? <p className="request-id">Request ID: {requestId}</p> : null}
        <button type="button" onClick={onRetry}>Try again</button>
      </div>
    </div>
  )
}

function formatCreatedAt(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}
