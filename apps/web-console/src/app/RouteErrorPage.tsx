import { isRouteErrorResponse, Link, useRouteError } from 'react-router-dom'

export function RouteErrorPage() {
  const error = useRouteError()
  const message = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText}`
    : error instanceof Error
      ? error.message
      : 'The requested page could not be displayed.'

  return (
    <section className="empty-state">
      <p className="eyebrow">Navigation error</p>
      <h1>Something went off course.</h1>
      <p>{message}</p>
      <Link className="button-link" to="/">
        Return to environments
      </Link>
    </section>
  )
}
