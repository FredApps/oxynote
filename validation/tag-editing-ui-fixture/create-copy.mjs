import { createHash } from 'node:crypto'
import { cpSync, existsSync, mkdirSync, readFileSync, realpathSync, symlinkSync, writeFileSync } from 'node:fs'
import { basename, dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const [sourceArg, targetArg] = process.argv.slice(2)
if (!sourceArg || !targetArg) throw new Error('Usage: node create-copy.mjs <original-web> <new-fixture-web>')
const source = realpathSync(sourceArg)
const target = resolve(targetArg)
const root = dirname(fileURLToPath(import.meta.url))
if (existsSync(target)) throw new Error(`Refusing to overwrite ${target}`)
if (!existsSync(join(source, 'node_modules'))) throw new Error('Install normal frontend dependencies first')
const expected = JSON.parse(readFileSync(join(root, 'expected-web-files.json'), 'utf8'))
for (const [path, sha] of Object.entries(expected)) {
  if (createHash('sha256').update(readFileSync(join(source, path))).digest('hex') !== sha) throw new Error(`Source does not match #94 v3: ${path}`)
}
mkdirSync(target, { recursive: true })
const excluded = new Set(['node_modules', '.git', '.nuxt', '.output', '.env'])
cpSync(source, target, { recursive: true, filter: (path) => path === source || !excluded.has(basename(path)) && !basename(path).startsWith('.env.') })
symlinkSync(join(source, 'node_modules'), join(target, 'node_modules'), process.platform === 'win32' ? 'junction' : 'dir')
cpSync(join(root, 'files'), target, { recursive: true })
const config = join(target, 'nuxt.config.ts')
const text = readFileSync(config, 'utf8')
if (!text.includes('typeCheck: true')) throw new Error('Unexpected Nuxt typecheck setting; inspect the disposable config')
writeFileSync(config, text.replace('typeCheck: true', 'typeCheck: false'))
console.log(`Created isolated fixture: ${target}`)
