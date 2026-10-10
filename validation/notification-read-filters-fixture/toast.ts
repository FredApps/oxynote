import { state } from './runtime'

export function showToastMessage(type: string, title: string, description?: string) {
  state.toasts.push([type, title, description].filter(Boolean).join(': '))
}
