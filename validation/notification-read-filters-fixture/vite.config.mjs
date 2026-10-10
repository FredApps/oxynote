import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

const root = dirname(fileURLToPath(import.meta.url))
const imports = {
  'app/components/NotificationBox.vue': `
import { ref, computed, watch, onMounted, onUnmounted, useTemplateRef } from 'vue'
import { useInfiniteScroll, useNow } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import useNotificationAPI from '~/composables/api/useNotificationAPI'
import { useDocumentAPI, useAuthSession, useWebSocketStateStore, navigateTo } from '@fixture/runtime'
import { NotificationCode, WS_NOTIFICATION_CREATION_TOPIC } from '~/utils/api/notification'
import { createNameSlug, createNameSlugWithId } from '~/utils/url'
`,
  'app/components/NotificationRow.vue': `
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NotificationCode } from '~/utils/api/notification'
import { DocumentHookType } from '~/utils/api/document'
import { relativeTimeLabel } from '~/utils/time'
`,
  'app/components/notification.ts': `
import { NotificationCode } from '~/utils/api/notification'
import { DocumentHookType } from '~/utils/api/document'
import { displayURL } from '~/utils/url'
`,
  'app/composables/api/useNotificationAPI.ts': `
import { useQueryCache, useQuery, useInfiniteQuery, useMutation } from '@pinia/colada'
import { toValue } from 'vue'
import { useNuxtApp } from '@fixture/runtime'
import { clone } from '~/utils/object'
`,
  'app/utils/object.ts': `import { jsonStableStringify } from '~/utils/json'\n`,
}

export default defineConfig({
  root,
  base: './',
  resolve: {
    alias: {
      '~': resolve(root, 'subject/app'),
      '@': resolve(root, 'subject/app'),
      '@fixture': root,
    },
    dedupe: ['vue', 'pinia', '@pinia/colada'],
  },
  plugins: [
    {
      name: 'fixture-nuxt-auto-imports',
      enforce: 'pre',
      resolveId(id, importer) {
        if (id === './toast' && importer?.includes('/NotificationBox.vue')) {
          return resolve(root, 'toast.ts')
        }
      },
      transform(code, id) {
        const relative = id.split('/subject/')[1]
        const injected = imports[relative]
        if (!injected) return null
        return {
          code: relative.endsWith('.vue')
            ? code.replace(/(<script[^>]*>)/, `$1\n${injected}`)
            : injected + code,
          map: null,
        }
      },
    },
    vue(),
    tailwindcss(),
  ],
  define: {
    __FIXTURE_SOURCE__: JSON.stringify(JSON.parse(readFileSync(resolve(root, 'source-lock.json'), 'utf8'))),
    __VUE_I18N_FULL_INSTALL__: true,
    __VUE_I18N_LEGACY_API__: false,
    __INTLIFY_PROD_DEVTOOLS__: false,
  },
  server: { host: '127.0.0.1', port: 4173, strictPort: true },
  preview: { host: '127.0.0.1', port: 4173, strictPort: true },
})
