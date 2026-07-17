import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from './AppLayout'
import { RouteErrorPage } from './RouteErrorPage'
import { CreateEnvironmentPage } from '../pages/CreateEnvironmentPage'
import { DashboardPage } from '../pages/DashboardPage'
import { EnvironmentDetailsPage } from '../pages/EnvironmentDetailsPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'environments/new', element: <CreateEnvironmentPage /> },
      { path: 'environments/:environmentId', element: <EnvironmentDetailsPage /> },
    ],
  },
])
