import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { dirname, extname, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), 'dist')
const types = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.json': 'application/json; charset=utf-8' }
const server = createServer(async (request, response) => {
  try {
    const path = decodeURIComponent(new URL(request.url || '/', 'http://127.0.0.1').pathname)
    const file = resolve(root, `.${path === '/' ? '/index.html' : path}`)
    if (!file.startsWith(root + sep)) {
      response.writeHead(403).end()
      return
    }
    const content = await readFile(file)
    response.writeHead(200, { 'Content-Type': types[extname(file)] || 'application/octet-stream', 'Cache-Control': 'no-store' })
    response.end(content)
  } catch {
    if (!response.headersSent) response.writeHead(404)
    response.end('Not found')
  }
})
server.listen(4173, '127.0.0.1', () => console.log('Synthetic fixture only: http://127.0.0.1:4173. Stop with Ctrl+C.'))
