import { createReadStream } from 'node:fs'
import { stat } from 'node:fs/promises'
import { createServer, request as proxyRequest } from 'node:http'
import { extname, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const port = Number(process.env.PORT || 3000)
const apiTarget = new URL(process.env.API_PROXY_URL || 'http://control-plane:8080')
const distDirectory = resolve(fileURLToPath(new URL('./dist', import.meta.url)))

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.map': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
}

createServer(async (incoming, response) => {
  const requestURL = new URL(incoming.url || '/', `http://${incoming.headers.host || 'localhost'}`)

  if (requestURL.pathname === '/healthz') {
    response.writeHead(200, { 'Content-Type': 'text/plain; charset=utf-8' })
    response.end('ok')
    return
  }

  if (requestURL.pathname.startsWith('/api/')) {
    proxyAPI(incoming, response, requestURL)
    return
  }

  const requestedPath = decodeURIComponent(requestURL.pathname)
  const candidate = resolve(distDirectory, `.${requestedPath === '/' ? '/index.html' : requestedPath}`)
  const isInsideDist = candidate === distDirectory || candidate.startsWith(`${distDirectory}${sep}`)
  if (!isInsideDist) {
    response.writeHead(400)
    response.end('Bad request')
    return
  }

  if (await isFile(candidate)) {
    sendFile(response, candidate)
    return
  }

  if (requestedPath.startsWith('/assets/')) {
    response.writeHead(404)
    response.end('Not found')
    return
  }

  sendFile(response, resolve(distDirectory, 'index.html'))
}).listen(port, '0.0.0.0', () => {
  console.log(`EnvPilot web console listening on :${port}`)
})

function proxyAPI(incoming, response, requestURL) {
  const target = new URL(`${requestURL.pathname}${requestURL.search}`, apiTarget)
  const outgoing = proxyRequest(target, {
    method: incoming.method,
    headers: { ...incoming.headers, host: target.host },
  }, (proxied) => {
    response.writeHead(proxied.statusCode || 502, proxied.headers)
    proxied.pipe(response)
  })

  outgoing.on('error', (error) => {
    if (!response.headersSent) {
      response.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' })
    }
    response.end(JSON.stringify({ code: 'BAD_GATEWAY', message: error.message, requestId: 'web-proxy' }))
  })
  incoming.pipe(outgoing)
}

async function isFile(path) {
  try {
    return (await stat(path)).isFile()
  } catch {
    return false
  }
}

function sendFile(response, path) {
  response.writeHead(200, {
    'Content-Type': contentTypes[extname(path)] || 'application/octet-stream',
    'Cache-Control': path.includes(`${sep}assets${sep}`) ? 'public, max-age=31536000, immutable' : 'no-cache',
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
  })
  createReadStream(path).pipe(response)
}
