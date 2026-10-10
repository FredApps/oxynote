import { createApp, defineComponent, h } from 'vue'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import NotificationRow from './subject/app/components/NotificationRow.vue'
import Button from './subject/app/components/shadcn/ui/button/Button.vue'
import Avatar from './subject/app/components/shadcn/ui/avatar/Avatar.vue'
import AvatarImage from './subject/app/components/shadcn/ui/avatar/AvatarImage.vue'
import AvatarFallback from './subject/app/components/shadcn/ui/avatar/AvatarFallback.vue'
import coladaOptions from './subject/colada.options'
import notificationMessages from './subject/i18n/locales/en/notification.json'
import generalMessages from './subject/i18n/locales/en/general.json'
import './style.css'

const app = createApp(App)
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: { ...notificationMessages, ...generalMessages } },
  datetimeFormats: {
    en: {
      'month-day-short': { month: 'short', day: 'numeric' },
      'short-with-time': { month: 'short', day: 'numeric', hour: 'numeric', minute: 'numeric' },
    },
  },
})
app.use(createPinia())
app.use(PiniaColada, coladaOptions)
app.use(i18n)
app.component('ShadcnUiButton', Button)
app.component('ShadcnUiAvatar', Avatar)
app.component('ShadcnUiAvatarImage', AvatarImage)
app.component('ShadcnUiAvatarFallback', AvatarFallback)
app.component('LazyDefaultAvatar', defineComponent({
  setup() { return () => h('span', { 'aria-hidden': true }, 'QA') },
}))
app.component('Icon', defineComponent({
  props: ['name'],
  setup(props) {
    return () => h('span', { 'aria-hidden': true, 'data-fixture-icon': props.name },
      props.name?.includes('checks') ? '✓✓' : props.name?.includes('check') ? '✓' : props.name?.includes('caret') ? '«' : '•')
  },
}))
// the wrapper adds observation attributes without changing the source row.
app.component('NotificationRow', defineComponent({
  inheritAttrs: false,
  props: ['notification'],
  setup(props, { attrs }) {
    return () => h(NotificationRow, {
      ...attrs,
      notification: props.notification,
      'data-fixture-id': props.notification.id,
      'data-fixture-read': String(props.notification.read),
    })
  },
}))
app.config.errorHandler = (error, _instance, info) => {
  const output = document.createElement('pre')
  output.className = 'fixture-error'
  output.textContent = `Fixture runtime error (${info}): ${String(error)}`
  document.body.append(output)
  console.error(error)
}
app.mount('#app')
