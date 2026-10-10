import { reactive, ref } from 'vue'
import { NotificationCode, WS_NOTIFICATION_CREATION_TOPIC } from './subject/app/utils/api/notification'
import type { Notification } from './subject/app/utils/api/notification'

interface LogEntry {
  sequence: number
  method: string
  url: string
  body?: { ids: string[] }
  state: string
  rows?: number
  pageCount?: number
  count?: number
  ids?: string[]
}

const documentId = 'fixture-document0001'
const listeners = new Map<string, Set<() => void>>()
const held: Array<() => void> = []
let nextId = 127
export const state = reactive({
  rows: Array.from({ length: 126 }, (_, index) => makeRow(index + 1, index % 2 === 1)),
  log: [] as LogEntry[],
  holdNextList: false,
  heldCount: 0,
  failNextWrite: false,
  delayMs: 150,
  events: 0,
  navigations: [] as string[],
  toasts: [] as string[],
})
const documentTree = ref([{
  id: documentId,
  documentName: 'Synthetic QA Page',
  icon: '',
  protected: false,
  children: [],
}])
const organization = {
  data: { data: { slug: 'fixture-org', members: [{ userId: 'fixture-actor', user: { name: 'Fixture Author', image: null } }] } },
}

function makeRow(index: number, read: boolean): Notification {
  return {
    id: `fixture-${String(index).padStart(3, '0')}`,
    userId: 'fixture-owner',
    organizationId: 'fixture-org',
    code: NotificationCode.DocumentNewComment,
    metadata: {
      userId: 'fixture-actor',
      documentId,
      branchId: 'fixture-branch',
      commentId: `comment-${index}`,
      anchorBlockId: null,
      commentExcerpt: `Synthetic notification ${String(index).padStart(3, '0')}`,
    },
    read,
    createdAt: new Date(Date.now() - index * 30 * 60 * 1000).toISOString(),
  }
}

async function coreAPIClient(url: string, options: { method: string; body?: { ids: string[] } }) {
  const entry = reactive<LogEntry>({ sequence: state.log.length + 1, method: options.method, url, body: options.body, state: 'pending' })
  state.log.push(entry)
  const parsed = new URL(url, 'http://fixture.invalid')
  const isList = parsed.pathname === '/api/notifications' && options.method === 'GET'
  if (isList && state.holdNextList) {
    state.holdNextList = false
    entry.state = 'held'
    await new Promise<void>(resolve => {
      held.push(resolve)
      state.heldCount = held.length
    })
  }
  await new Promise(resolve => setTimeout(resolve, state.delayMs))

  if (options.method === 'PUT' && parsed.pathname === '/api/notifications/read-status') {
    if (state.failNextWrite) {
      state.failNextWrite = false
      entry.state = 'simulated 500'
      throw new Error('Synthetic write failure')
    }
    const ids = options.body?.ids || []
    state.rows.forEach(row => { if (ids.length === 0 || ids.includes(row.id)) row.read = true })
    entry.state = '200'
    return undefined
  }
  if (parsed.pathname === '/api/notifications/count' && options.method === 'GET') {
    const read = parsed.searchParams.get('read') === 'true'
    entry.count = state.rows.filter(row => row.read === read).length
    entry.state = '200'
    return { count: entry.count }
  }
  if (isList) {
    const filter = parsed.searchParams.get('filter-read_eq')
    const rows = state.rows.filter(row => filter === null || row.read === (filter === 'true'))
    const limit = Number(parsed.searchParams.get('limit'))
    const page = Number(parsed.searchParams.get('page'))
    const selected = rows.slice((page - 1) * limit, page * limit)
    entry.rows = selected.length
    entry.pageCount = Math.ceil(rows.length / limit)
    entry.ids = selected.map(row => row.id)
    entry.state = '200'
    return { notifications: JSON.parse(JSON.stringify(selected)), pageCount: entry.pageCount }
  }
  entry.state = 'unexpected request'
  throw new Error(`Unexpected synthetic request: ${options.method} ${url}`)
}

export function useNuxtApp() { return { $coreAPIClient: coreAPIClient } }
export function useDocumentAPI() { return { fetchDocumentTree: { data: documentTree } } }
export function useAuthSession() {
  return { fetchOrganization: { state: ref(organization), refresh: async () => organization } }
}
export function useWebSocketStateStore() {
  return {
    state: {
      subscribe(topic: string, callback: () => void) {
        if (!listeners.has(topic)) listeners.set(topic, new Set())
        listeners.get(topic)!.add(callback)
        return () => listeners.get(topic)?.delete(callback)
      },
    },
  }
}
export async function navigateTo(href: string) { state.navigations.push(href) }
export function emitCreation() {
  state.events += 1
  listeners.get(WS_NOTIFICATION_CREATION_TOPIC)?.forEach(callback => callback())
}
export function addLiveRow(read = false) {
  const row = makeRow(nextId++, read)
  row.createdAt = new Date().toISOString()
  state.rows.unshift(row)
  emitCreation()
}
export function releaseHeld() {
  held.splice(0).forEach(release => release())
  state.heldCount = 0
}
