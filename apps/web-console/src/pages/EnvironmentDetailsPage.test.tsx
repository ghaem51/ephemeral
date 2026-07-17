import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  destroyEnvironment,
  getEnvironment,
  retryEnvironment,
  type Environment,
  type EnvironmentStatus,
  type WorkflowStatus,
} from '../api/environments'
import { EnvironmentDetailsPage } from './EnvironmentDetailsPage'

vi.mock('../api/environments', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/environments')>()
  return {
    ...actual,
    getEnvironment: vi.fn(),
    retryEnvironment: vi.fn(),
    destroyEnvironment: vi.fn(),
  }
})

describe('EnvironmentDetailsPage', () => {
  beforeEach(() => {
    vi.mocked(getEnvironment).mockReset()
    vi.mocked(retryEnvironment).mockReset()
    vi.mocked(destroyEnvironment).mockReset()
  })

  it('renders ordered workflow step state, messages, errors, timing, and duration', async () => {
    vi.mocked(getEnvironment).mockResolvedValue(environmentFixture('FAILED', 'FAILED'))

    renderPage()

    expect(await screen.findByRole('heading', { name: 'Create workflow' })).toBeInTheDocument()
    const steps = screen.getAllByRole('listitem')
    expect(steps).toHaveLength(5)
    expect(within(steps[0]).getByRole('heading', { name: 'Validate Request' })).toBeInTheDocument()
    expect(within(steps[0]).getByText('Succeeded')).toBeInTheDocument()
    expect(within(steps[0]).getByText('2s')).toBeInTheDocument()
    expect(within(steps[1]).getByText('Running')).toBeInTheDocument()
    expect(within(steps[2]).getByText('container exited')).toBeInTheDocument()
    expect(within(steps[2]).getByText('Failed')).toBeInTheDocument()
    expect(within(steps[3]).getByText('Skipped')).toBeInTheDocument()
    expect(within(steps[4]).getByText('Pending')).toBeInTheDocument()
  })

  it.each([
    { status: 'READY', workflow: 'SUCCEEDED', open: true, retry: false, destroy: true },
    { status: 'FAILED', workflow: 'FAILED', open: false, retry: true, destroy: true },
    { status: 'PROVISIONING', workflow: 'RUNNING', open: false, retry: false, destroy: false },
    { status: 'DESTROYED', workflow: 'SUCCEEDED', open: false, retry: false, destroy: false },
  ] satisfies Array<{
    status: EnvironmentStatus
    workflow: WorkflowStatus
    open: boolean
    retry: boolean
    destroy: boolean
  }>)('shows valid actions for $status', async ({ status, workflow, open, retry, destroy }) => {
    vi.mocked(getEnvironment).mockResolvedValue(environmentFixture(status, workflow))

    renderPage()
    await screen.findByRole('heading', { name: 'feature-payment' })

    expect(Boolean(screen.queryByRole('link', { name: /Open environment/ }))).toBe(open)
    expect(Boolean(screen.queryByRole('button', { name: 'Retry' }))).toBe(retry)
    expect(Boolean(screen.queryByRole('button', { name: 'Destroy' }))).toBe(destroy)
  })
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/environments/env-1']}>
        <Routes><Route path="/environments/:environmentId" element={<EnvironmentDetailsPage />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function environmentFixture(status: EnvironmentStatus, workflowStatus: WorkflowStatus): Environment {
  const startedAt = '2026-07-16T12:00:00Z'
  return {
    id: 'env-1',
    name: 'feature-payment',
    image: 'envpilot/demo-service:healthy',
    containerPort: 8080,
    hostPort: 49152,
    containerId: '1234567890abcdef',
    url: 'http://localhost:49152',
    status,
    errorMessage: status === 'FAILED' ? 'container exited' : undefined,
    createdAt: startedAt,
    updatedAt: '2026-07-16T12:00:10Z',
    latestWorkflow: {
      id: 'workflow-1',
      environmentId: 'env-1',
      operation: 'CREATE',
      status: workflowStatus,
      startedAt,
      completedAt: workflowStatus === 'RUNNING' ? null : '2026-07-16T12:00:10Z',
      steps: [
        step('step-1', 1, 'VALIDATE_REQUEST', 'SUCCEEDED', startedAt, '2026-07-16T12:00:02Z'),
        step('step-2', 2, 'CREATE_CONTAINER', 'RUNNING', startedAt, null),
        { ...step('step-3', 3, 'START_CONTAINER', 'FAILED', startedAt, '2026-07-16T12:00:03Z'), errorMessage: 'container exited' },
        step('step-4', 4, 'CHECK_HEALTH', 'SKIPPED', null, null),
        step('step-5', 5, 'MARK_READY', 'PENDING', null, null),
      ],
    },
  }
}

function step(
  id: string,
  order: number,
  name: string,
  status: 'PENDING' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'SKIPPED',
  startedAt: string | null,
  completedAt: string | null,
) {
  return { id, workflowId: 'workflow-1', name, order, status, message: `${name} message`, startedAt, completedAt }
}
