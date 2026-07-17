import type { StepStatus } from '../api/environments'

const labels: Record<StepStatus, string> = {
  PENDING: 'Pending',
  RUNNING: 'Running',
  SUCCEEDED: 'Succeeded',
  FAILED: 'Failed',
  SKIPPED: 'Skipped',
}

export function StepStatusBadge({ status }: { status: StepStatus }) {
  return <span className={`status-badge status-${status.toLowerCase()}`}>{labels[status]}</span>
}
