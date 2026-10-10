import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { copyFile, mkdir, readFile, symlink, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const repo = resolve(process.argv[2] || '../oxynote-notifications')
const expectedCommit = process.argv[3]
const allowDirty = process.argv.includes('--allow-dirty')
const git = (...args) => execFileSync('git', ['-C', repo, ...args], { encoding: 'utf8' }).trim()
const commit = git('rev-parse', 'HEAD')
if (!expectedCommit || commit !== expectedCommit) {
  throw new Error(`Supply the exact 40-character source commit. Checkout is ${commit}`)
}
const paths = [
  'app/components/NotificationBox.vue',
  'app/components/NotificationRow.vue',
  'app/components/notification.ts',
  'app/components/sidebar.ts',
  'app/components/shadcn/ui/button/Button.vue',
  'app/components/shadcn/ui/button/index.ts',
  'app/components/shadcn/ui/avatar/Avatar.vue',
  'app/components/shadcn/ui/avatar/AvatarImage.vue',
  'app/components/shadcn/ui/avatar/AvatarFallback.vue',
  'app/composables/api/useNotificationAPI.ts',
  'app/utils/api/notification.ts',
  'app/utils/api/document.ts',
  'app/utils/object.ts',
  'app/utils/json.ts',
  'app/utils/time.ts',
  'app/utils/url.ts',
  'app/lib/utils.ts',
  'i18n/locales/en/notification.json',
  'i18n/locales/en/general.json',
  'colada.options.ts',
]
const dirty = git('status', '--porcelain', '--', ...paths.map(path => `web/${path}`))
if (dirty && !allowDirty) {
  throw new Error('Subject files are modified. Commit them first; --allow-dirty is preparation-only.')
}
const hashes = {}
for (const path of paths) {
  const source = resolve(repo, 'web', path)
  const destination = resolve(here, 'subject', path)
  await mkdir(dirname(destination), { recursive: true })
  await copyFile(source, destination)
  hashes[`web/${path}`] = createHash('sha256').update(await readFile(source)).digest('hex')
}
const manifest = {
  sourceRepository: 'oxynote/oxynote',
  sourceCommit: commit,
  sourceDirty: Boolean(dirty),
  boundAt: new Date().toISOString(),
  files: hashes,
}
await writeFile(resolve(here, 'source-lock.json'), JSON.stringify(manifest, null, 2) + '\n')
const dependencyPath = resolve(here, 'node_modules')
try { await symlink(resolve(repo, 'web/node_modules'), dependencyPath, 'junction') }
catch (error) { if (error.code !== 'EEXIST') throw error }
console.log(JSON.stringify(manifest, null, 2))
