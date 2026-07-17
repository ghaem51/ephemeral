const configuredBaseURL = import.meta.env.VITE_API_BASE_URL?.trim()

export const apiBaseURL = (configuredBaseURL || '').replace(/\/$/, '')

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
    readonly requestId?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

type ErrorPayload = {
  code?: string
  message?: string
  requestId?: string
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseURL}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  if (!response.ok) {
    const payload = await readErrorPayload(response)
    throw new ApiError(
      payload.message || `Request failed with status ${response.status}`,
      response.status,
      payload.code,
      payload.requestId,
    )
  }

  return (await response.json()) as T
}

async function readErrorPayload(response: Response): Promise<ErrorPayload> {
  try {
    return (await response.json()) as ErrorPayload
  } catch {
    return {}
  }
}
