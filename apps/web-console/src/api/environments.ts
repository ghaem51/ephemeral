import { apiRequest } from './client'

export type EnvironmentStatus =
  | 'PENDING'
  | 'PROVISIONING'
  | 'READY'
  | 'FAILED'
  | 'DESTROYING'
  | 'DESTROYED'

export type WorkflowOperation = 'CREATE' | 'DESTROY' | 'RETRY'

export type WorkflowStep = {
  id: string
  workflowId: string
  name: string
  order: number
  status: string
  message: string
  errorMessage?: string
  startedAt: string | null
  completedAt: string | null
}

export type Workflow = {
  id: string
  environmentId: string
  operation: WorkflowOperation
  status: string
  startedAt: string | null
  completedAt: string | null
  steps: WorkflowStep[]
}

export type Environment = {
  id: string
  name: string
  image: string
  containerPort: number
  hostPort: number
  containerId: string
  url: string
  status: EnvironmentStatus
  errorMessage?: string
  createdAt: string
  updatedAt: string
  latestWorkflow: Workflow | null
}

export type CreateEnvironmentInput = {
  name: string
  image: 'envpilot/demo-service:healthy'
  containerPort: 8080
  simulateFailure: boolean
  applicationVersion?: string
}

export function listEnvironments() {
  return apiRequest<Environment[]>('/api/v1/environments')
}

export function createEnvironment(input: CreateEnvironmentInput) {
  return apiRequest<Environment>('/api/v1/environments', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function isActiveEnvironment(environment: Environment) {
  return ['PENDING', 'PROVISIONING', 'DESTROYING'].includes(environment.status)
}
