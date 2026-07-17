import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { AppErrorBoundary } from './app/AppErrorBoundary'
import { AppQueryProvider } from './app/AppQueryProvider'
import { router } from './app/router'
import './styles.css'

const root = document.getElementById('root')

if (!root) {
  throw new Error('Application root element was not found')
}

createRoot(root).render(
  <StrictMode>
    <AppErrorBoundary>
      <AppQueryProvider>
        <RouterProvider router={router} />
      </AppQueryProvider>
    </AppErrorBoundary>
  </StrictMode>,
)
