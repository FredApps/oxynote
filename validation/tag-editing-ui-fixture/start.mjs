import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { join, resolve } from 'node:path'
const root = resolve(process.argv[2] ?? '')
const port = Number(process.argv[3] ?? 3344)
if (!process.argv[2] || !existsSync(join(root,'app/pages/tag-edit-validation.vue'))) throw new Error('Usage: node start.mjs <fixture-web> [port]')
if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('Use an unprivileged local port')
const origin = `http://127.0.0.1:${port}`
const api = origin + '/__tag_fixture'
const env = {...process.env, DESKTOP_BUILD:'0', NITRO_PRESET:'node-server',
 NUXT_PUBLIC_APP_BASE_URL:origin,
 NUXT_PUBLIC_CORE_API_BASE_HTTP_URL:api,
 NUXT_PUBLIC_AUTH_REALTIME_API_BASE_HTTP_URL:api,
 NUXT_CORE_API_INTERNAL_HTTP_URL:api,
 NUXT_AUTH_REALTIME_API_INTERNAL_HTTP_URL:api,
 NUXT_PUBLIC_CORE_API_BASE_WS_URL:`ws://127.0.0.1:${port}/__tag_fixture`,
 NUXT_PUBLIC_AUTH_REALTIME_API_BASE_WS_URL:`ws://127.0.0.1:${port}/__tag_fixture`
}
console.log(`Open ${origin}/tag-edit-validation in the existing Chrome tab`)
const child = spawn(process.execPath,[join(root,'node_modules/@nuxt/cli/bin/nuxi.mjs'),'dev','--host','127.0.0.1','--port',String(port),'--no-fork'],{cwd:root,env,stdio:'inherit'})
process.on('SIGINT',()=>child.kill('SIGINT'))
process.on('SIGTERM',()=>child.kill('SIGTERM'))
child.on('exit',code=>{process.exitCode=code??1})
