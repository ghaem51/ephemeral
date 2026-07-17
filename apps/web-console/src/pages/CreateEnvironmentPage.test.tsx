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
    ['', 'Enter an environment name.'],
    ['Feature_Payment', 'Use only lowercase letters, numbers, and hyphens. Start and end with a letter or number.'],
    ['-leading-hyphen', 'Use only lowercase letters, numbers, and hyphens. Start and end with a letter or number.'],
  ])('shows an error and does not submit invalid name %j', async (name, expectedError) => {
    const user = userEvent.setup()
    renderPage()
    const input = screen.getByLabelText('Environment name')

    if (name) await user.type(input, name)
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(input).toBeInvalid()
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('alert')).toHaveTextContent(expectedError)
    expect(createEnvironment).not.toHaveBeenCalled()
  })

  it('clears the environment name error when the user edits the field', async () => {
    const user = userEvent.setup()
    renderPage()
    const input = screen.getByLabelText('Environment name')

    await user.type(input, 'Invalid_Name')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))
    expect(screen.getByRole('alert')).toBeInTheDocument()

    await user.clear(input)
    await user.type(input, 'valid-name')

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(input).not.toHaveAttribute('aria-invalid')
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
    await user.type(screen.getByLabelText(/Environment variables/), 'API_URL=https://api.example.test{enter}FEATURE_FLAG=true')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(vi.mocked(createEnvironment).mock.calls[0]?.[0]).toEqual({
      name: 'custom-service',
      image: 'nginx:latest',
      containerPort: 80,
      healthCheckPath: '/ready',
      environmentVariables: ['API_URL=https://api.example.test', 'FEATURE_FLAG=true'],
      simulateFailure: false,
    })
  })

  it('does not submit invalid or reserved environment variables', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Environment name'), 'invalid-env')
    await user.type(screen.getByLabelText(/Environment variables/), 'APP_VERSION=override')
    await user.click(screen.getByRole('button', { name: 'Create environment' }))

    expect(createEnvironment).not.toHaveBeenCalled()
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
