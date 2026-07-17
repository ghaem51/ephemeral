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
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter><CreateEnvironmentPage /></MemoryRouter>
    </QueryClientProvider>,
  )
}
