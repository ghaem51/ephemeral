import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { EnvironmentStatus, StepStatus } from '../api/environments'
import { StatusBadge } from './StatusBadge'
import { StepStatusBadge } from './StepStatusBadge'

describe('status badges', () => {
  it.each<EnvironmentStatus>(['PENDING', 'PROVISIONING', 'READY', 'FAILED', 'DESTROYING', 'DESTROYED'])(
    'renders the %s environment state',
    (status) => {
      render(<StatusBadge status={status} />)

      expect(screen.getByText(status === 'PROVISIONING' ? 'Provisioning' : titleCase(status))).toHaveClass(
        `status-${status.toLowerCase()}`,
      )
    },
  )

  it.each<StepStatus>(['PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'SKIPPED'])(
    'renders the %s workflow-step state',
    (status) => {
      render(<StepStatusBadge status={status} />)

      expect(screen.getByText(titleCase(status))).toHaveClass(`status-${status.toLowerCase()}`)
    },
  )
})

function titleCase(value: string) {
  return value.charAt(0) + value.slice(1).toLowerCase()
}

