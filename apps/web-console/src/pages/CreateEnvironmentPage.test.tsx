import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createEnvironment } from '../api/environments'
import { CreateEnvironmentPage } from './CreateEnvironmentPage'

vi.mock('../api/environments', () => ({
  createEnvironment: vi.fn(),
}))

describe('CreateEnvironmentPage validation', () => {
  beforeEach(() => vi.mocked(createEnvironment).mockReset())

  it.each([
    ['', 'a value is required'],
    ['Feature_Payment', 'lowercase letters, numbers, and hyphens'],
    ['-leading-hyphen', 'lowercase letters, numbers, and hyphens'],
  ])('does not submit invalid name %j', async (name) => {
    const user = userEvent.setup()
    renderPage()
    const input = screen.getByLabelText('Environment name')

    if (name) await user.type(input, name)
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    if (!name) expect(input).toBeInvalid()
    else expect(input).toHaveAttribute('pattern', '[a-z0-9](?:[a-z0-9-]*[a-z0-9])?')
    expect(createEnvironment).not.toHaveBeenCalled()
  })

  it('requires the optional version to use the supported format', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Environment name'), 'feature-payment')
    const version = screen.getByLabelText(/Application version/)
    await user.type(version, 'release candidate')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(version).toHaveAttribute('pattern', '[A-Za-z0-9][A-Za-z0-9._-]{0,63}')
    expect(createEnvironment).not.toHaveBeenCalled()
  })

  it('submits a custom image and container port', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Environment name'), 'custom-service')
    await user.click(screen.getByRole('radio', { name: /Custom Docker image/ }))
    await user.type(screen.getByLabelText('Container image'), 'nginx:latest')
    await user.clear(screen.getByLabelText('Container port'))
    await user.type(screen.getByLabelText('Container port'), '80')
    await user.clear(screen.getByLabelText('Health check path'))
    await user.type(screen.getByLabelText('Health check path'), '/ready')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(vi.mocked(createEnvironment).mock.calls[0]?.[0]).toEqual({
      name: 'custom-service',
      image: 'nginx:latest',
      containerPort: 80,
      healthCheckPath: '/ready',
      simulateFailure: false,
    })
  })

  it('rejects a health check path without a leading slash', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Environment name'), 'invalid-health-path')
    const path = screen.getByLabelText('Health check path')
    await user.clear(path)
    await user.type(path, 'ready')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(path).toBeInvalid()
    expect(createEnvironment).not.toHaveBeenCalled()
  })
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter><CreateEnvironmentPage /></MemoryRouter>
    </QueryClientProvider>,
  )
}
