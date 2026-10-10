<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useQueryCache } from '@pinia/colada'
import NotificationBox from './subject/app/components/NotificationBox.vue'
import { state, addLiveRow, emitCreation, releaseHeld } from './runtime'

const source = __FIXTURE_SOURCE__
const cache = useQueryCache()
const subjectHost = ref<HTMLElement>()
const open = ref(true)
const rendered = ref({ all: 0, unread: 0, read: 0, ids: [] as string[] })
const cacheSnapshot = ref('Use “Inspect cache” to capture actual Colada entries.')
const unread = computed(() => state.rows.filter(row => !row.read).length)
let observer: MutationObserver | undefined

onMounted(() => {
  observer = new MutationObserver(inspectRendered)
  observer.observe(subjectHost.value!, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-fixture-read'] })
  inspectRendered()
  Object.assign(window, {
    notificationFixture: {
      state,
      source,
      inspectCache,
      rendered,
      addLiveRow,
      emitCreation,
      releaseHeld,
    },
  })
})
onUnmounted(() => observer?.disconnect())

function inspectRendered() {
  const rows = [...(subjectHost.value?.querySelectorAll<HTMLElement>('[data-fixture-id]') || [])]
  rendered.value = {
    all: rows.length,
    unread: rows.filter(row => row.dataset.fixtureRead === 'false').length,
    read: rows.filter(row => row.dataset.fixtureRead === 'true').length,
    ids: rows.map(row => row.dataset.fixtureId!),
  }
}
function inspectCache() {
  const snapshot = cache.getEntries({ key: ['notifications'] }).map(entry => ({
    key: entry.key,
    data: cache.getQueryData(entry.key),
  }))
  cacheSnapshot.value = JSON.stringify(snapshot, null, 2)
  return snapshot
}
function reset() { window.location.reload() }
function clearLog() { state.log.splice(0) }
</script>

<template>
  <main class="fixture-layout">
    <section class="fixture-intro">
      <h1>Oxynote #201: synthetic browser QA</h1>
      <p>Actual NotificationBox, NotificationRow, useNotificationAPI, Pinia Colada and VueUse. In-memory API, synthetic account and event bus. This is not backend or full-app end-to-end evidence.</p>
      <p class="fixture-source">Source: {{ source.sourceCommit }} · {{ source.sourceDirty ? 'UNCOMMITTED PREPARATION SNAPSHOT' : 'committed source' }}</p>
    </section>

    <div ref="subjectHost" class="fixture-subject">
      <NotificationBox v-if="open" @close-notification-box="open = false" />
      <button v-else class="fixture-button" @click="open = true">Reopen notifications</button>
    </div>

    <section class="fixture-controls">
      <h2>Fixture controls and evidence</h2>
      <p data-testid="server-counts">Synthetic store: {{ state.rows.length }} total · {{ unread }} unread · {{ state.rows.length - unread }} read</p>
      <p data-testid="rendered-counts">Rendered: {{ rendered.all }} rows · {{ rendered.unread }} unread · {{ rendered.read }} read</p>
      <p>Scroll the notification list to load the next page. Each filter has its own cache and server-side result set.</p>
      <div class="fixture-actions">
        <button class="fixture-button" @click="addLiveRow(false)">Add live unread</button>
        <button class="fixture-button" @click="addLiveRow(true)">Add live read</button>
        <button class="fixture-button" @click="emitCreation">Emit creation event only</button>
        <button class="fixture-button" @click="reset">Reset 126 rows and cache</button>
      </div>
      <div class="fixture-actions">
        <label><input v-model="state.holdNextList" type="checkbox"> Hold next list response</label>
        <button class="fixture-button" :disabled="!state.heldCount" @click="releaseHeld">Release held ({{ state.heldCount }})</button>
        <label><input v-model="state.failNextWrite" type="checkbox"> Fail next mark-read write</label>
        <label>Latency (ms) <input v-model.number="state.delayMs" type="number" min="0" max="5000" step="100"></label>
      </div>
      <p>Creation events sent: {{ state.events }}. Row clicks log the destination below and mark read; the fixture stays on this page.</p>
      <p v-if="state.navigations.length">Last intercepted navigation: {{ state.navigations.at(-1) }}</p>
      <p v-for="(toast, index) in state.toasts" :key="index" role="alert">{{ toast }}</p>

      <h3>Transport request log</h3>
      <p>All: omit filter-read_eq. Unread: filter-read_eq=false. Read: filter-read_eq=true. Pagination keeps that parameter.</p>
      <button class="fixture-button" @click="clearLog">Clear request log</button>
      <div class="fixture-log" data-testid="request-log">
        <div v-for="entry in [...state.log].reverse()" :key="entry.sequence" class="fixture-log-entry">
          <strong>#{{ entry.sequence }} {{ entry.method }} {{ entry.url }}</strong>
          <div>{{ entry.state }}<template v-if="entry.rows !== undefined"> · rows={{ entry.rows }} · pageCount={{ entry.pageCount }}</template><template v-if="entry.count !== undefined"> · count={{ entry.count }}</template><template v-if="entry.body"> · body={{ JSON.stringify(entry.body) }}</template></div>
          <details v-if="entry.ids"><summary>Returned IDs</summary>{{ entry.ids.join(', ') }}</details>
        </div>
      </div>
      <details>
        <summary>Actual rendered IDs and Colada cache</summary>
        <p>{{ rendered.ids.join(', ') }}</p>
        <button class="fixture-button" @click="inspectCache">Inspect cache</button>
        <pre>{{ cacheSnapshot }}</pre>
      </details>
    </section>
  </main>
</template>
