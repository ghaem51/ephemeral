import { useParams } from 'react-router-dom'

export function EnvironmentDetailsPage() {
  const { environmentId } = useParams()

  return (
    <section>
      <div className="page-heading">
        <div>
          <p className="eyebrow">Environment details</p>
          <h1>{environmentId || 'Environment'}</h1>
          <p>Runtime information and workflow progress will appear here.</p>
        </div>
      </div>
      <div className="placeholder-card">
        <h2>Workflow details placeholder</h2>
        <p>Ordered provisioning steps, status, retry, and destroy actions will be added next.</p>
      </div>
    </section>
  )
}
