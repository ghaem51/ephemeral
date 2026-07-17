import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../api/client'
import {
  destroyEnvironment,
  getEnvironment,
  isActiveEnvironment,
  isActiveWorkflow,
  retryEnvironment,
  type WorkflowStep,
} from '../api/environments'
import { StatusBadge } from '../components/StatusBadge'
import { StepStatusBadge } from '../components/StepStatusBadge'

type LifecycleAction = 'retry' | 'destroy'

export function EnvironmentDetailsPage() {
  const { environmentId = '' } = useParams()
  const queryClient = useQueryClient()
  const environment = useQuery({
    queryKey: ['environment', environmentId],
    queryFn: () => getEnvironment(environmentId),
    enabled: environmentId !== '',
    refetchInterval: (query) => {
      const data = query.state.data
      return data && (isActiveEnvironment(data) || isActiveWorkflow(data)) ? 1_500 : false
    },
  })
  const action = useMutation({
    mutationFn: (selected: LifecycleAction) =>
      selected === 'retry' ? retryEnvironment(environmentId) : destroyEnvironment(environmentId),
    onSuccess: async (updated) => {
      queryClient.setQueryData(['environment', environmentId], updated)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['environment', environmentId] }),
        queryClient.invalidateQueries({ queryKey: ['environments'] }),
      ])
    },
  })

  if (environment.isPending) return <DetailsLoading />
  if (environment.isError) {
    if (environment.error instanceof ApiError && environment.error.status === 404) {
      return <DetailsNotFound />
    }
    return <DetailsError error={environment.error} onRetry={() => void environment.refetch()} />
  }

  const data = environment.data
  const workflowActive = isActiveEnvironment(data) || isActiveWorkflow(data)

  function confirmDestroy() {
    if (window.confirm(`Destroy environment “${data.name}”? This removes its runtime container.`)) {
      action.mutate('destroy')
    }
  }

  return (
    <section>
      <div className="page-heading details-heading">
        <div>
          <p className="eyebrow">Environment details</p>
          <div className="title-with-status"><h1>{data.name}</h1><StatusBadge status={data.status} /></div>
          <p>Created {formatTimestamp(data.createdAt)}</p>
        </div>
        <div className="action-bar">
          {data.status === 'READY' && data.url ? (
            <a className="button-link" href={data.url} target="_blank" rel="noreferrer">Open environment ↗</a>
          ) : null}
          {data.status === 'FAILED' ? (
            <button type="button" onClick={() => action.mutate('retry')} disabled={workflowActive || action.isPending}>
              {action.isPending && action.variables === 'retry' ? 'Retrying…' : 'Retry'}
            </button>
          ) : null}
          {data.status === 'READY' || data.status === 'FAILED' ? (
            <button className="danger-button" type="button" onClick={confirmDestroy} disabled={workflowActive || action.isPending}>
              {action.isPending && action.variables === 'destroy' ? 'Destroying…' : 'Destroy'}
            </button>
          ) : null}
        </div>
      </div>

      {action.isError ? <ActionError error={action.error} /> : null}

      <div className="details-grid">
        <section className="details-card runtime-card">
          <div className="card-heading"><div><p className="eyebrow">Runtime</p><h2>Environment information</h2></div></div>
          <dl className="metadata-grid">
            <Metadata label="Image"><code>{data.image}</code></Metadata>
            {data.applicationVersion ? <Metadata label="Application version"><code>{data.applicationVersion}</code></Metadata> : null}
            <Metadata label="Environment URL">
              {data.url ? <a href={data.url} target="_blank" rel="noreferrer">{data.url} ↗</a> : 'Not available'}
            </Metadata>
            <Metadata label="Container ID"><code title={data.containerId}>{shortContainerID(data.containerId)}</code></Metadata>
            <Metadata label="Ports">Host {data.hostPort || '—'} → Container {data.containerPort}</Metadata>
            <Metadata label="Created">{formatTimestamp(data.createdAt)}</Metadata>
            <Metadata label="Updated">{formatTimestamp(data.updatedAt)}</Metadata>
          </dl>
          {data.errorMessage ? (
            <div className="latest-error" role="alert"><strong>Latest error</strong><p>{data.errorMessage}</p></div>
          ) : null}
        </section>

        <section className="details-card workflow-card">
          <div className="card-heading workflow-heading">
            <div>
              <p className="eyebrow">Workflow timeline</p>
              <h2>{data.latestWorkflow ? `${titleCase(data.latestWorkflow.operation)} workflow` : 'No workflow recorded'}</h2>
            </div>
            {data.latestWorkflow ? <span className="workflow-state">{titleCase(data.latestWorkflow.status)}</span> : null}
          </div>
          {data.latestWorkflow ? (
            <ol className="workflow-timeline">
              {data.latestWorkflow.steps.map((step) => <TimelineStep key={step.id} step={step} />)}
            </ol>
          ) : (
            <p className="muted-copy">Workflow steps will appear after an operation begins.</p>
          )}
        </section>
      </div>
    </section>
  )
}

function TimelineStep({ step }: { step: WorkflowStep }) {
  return (
    <li className={`timeline-step timeline-${step.status.toLowerCase()}`}>
      <div className="timeline-marker"><span>{step.order}</span></div>
      <div className="timeline-content">
        <div className="timeline-title"><h3>{titleCase(step.name)}</h3><StepStatusBadge status={step.status} /></div>
        <p className="step-message">{step.message || defaultStepMessage(step.status)}</p>
        {step.errorMessage ? <p className="step-error">{step.errorMessage}</p> : null}
        <dl className="step-timing">
          <div><dt>Started</dt><dd>{formatOptionalTimestamp(step.startedAt)}</dd></div>
          <div><dt>Completed</dt><dd>{formatOptionalTimestamp(step.completedAt)}</dd></div>
          <div><dt>Duration</dt><dd>{formatDuration(step)}</dd></div>
        </dl>
      </div>
    </li>
  )
}

function Metadata({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><dt>{label}</dt><dd>{children}</dd></div>
}

function DetailsLoading() {
  return <div className="details-skeleton" role="status"><span className="spinner" /><div><h1>Loading environment</h1><p>Retrieving runtime and workflow state.</p></div></div>
}

function DetailsNotFound() {
  return <div className="empty-state"><p className="eyebrow">Environment not found</p><h1>This environment does not exist.</h1><p>It may have been removed or the URL may be incorrect.</p><Link className="button-link" to="/">Return to dashboard</Link></div>
}

function DetailsError({ error, onRetry }: { error: Error; onRetry: () => void }) {
  const requestId = error instanceof ApiError ? error.requestId : undefined
  return <div className="empty-state error-state" role="alert"><p className="eyebrow">API error</p><h1>Environment details are unavailable.</h1><p>{error.message}</p>{requestId ? <p className="request-id">Request ID: {requestId}</p> : null}<button type="button" onClick={onRetry}>Try again</button></div>
}

function ActionError({ error }: { error: Error }) {
  const requestId = error instanceof ApiError ? error.requestId : undefined
  return <div className="form-error action-error" role="alert"><strong>Action could not be completed</strong><p>{error.message}</p>{requestId ? <small>Request ID: {requestId}</small> : null}</div>
}

function shortContainerID(value: string) {
  return value ? value.slice(0, 12) : 'Not available'
}

function formatTimestamp(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
}

function formatOptionalTimestamp(value: string | null) {
  return value ? new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' }).format(new Date(value)) : '—'
}

function formatDuration(step: WorkflowStep) {
  if (!step.startedAt) return '—'
  const end = step.completedAt ? new Date(step.completedAt).getTime() : Date.now()
  const seconds = Math.max(0, Math.round((end - new Date(step.startedAt).getTime()) / 1_000))
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

function titleCase(value: string) {
  return value.toLowerCase().replaceAll('_', ' ').replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function defaultStepMessage(status: WorkflowStep['status']) {
  if (status === 'PENDING') return 'Waiting for the previous step.'
  if (status === 'RUNNING') return 'Operation in progress.'
  if (status === 'SKIPPED') return 'This step was not required.'
  return 'Step completed.'
}
