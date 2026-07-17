import type { EnvironmentStatus } from '../api/environments'

const labels: Record<EnvironmentStatus, string> = {
  PENDING: 'Pending',
  PROVISIONING: 'Provisioning',
  READY: 'Ready',
  FAILED: 'Failed',
  DESTROYING: 'Destroying',
  DESTROYED: 'Destroyed',
}

export function StatusBadge({ status }: { status: EnvironmentStatus }) {
  return <span className={`status-badge status-${status.toLowerCase()}`}>{labels[status]}</span>
}
