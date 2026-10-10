import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { Window } from 'happy-dom'

const window = new Window({ url: 'http://127.0.0.1:4173' })
const file = readdirSync(new URL('./dist/assets/', import.meta.url)).find(name => name.endsWith('.js'))
window.document.body.innerHTML = '<div id="app"></div>'
window.eval(readFileSync(new URL(`./dist/assets/${file}`, import.meta.url), 'utf8'))
const button = label => [...window.document.querySelectorAll('button')].find(element => element.textContent.trim() === label)
const rows = () => [...window.document.querySelectorAll('[data-fixture-id]')]
const waitFor = async predicate => {
  const deadline = Date.now() + 3000
  while (!predicate()) {
    assert.ok(Date.now() < deadline, 'Synthetic DOM smoke timed out')
    await new Promise(resolve => setTimeout(resolve, 10))
  }
}
const settled = () => !window.notificationFixture.state.log.some(entry => ['pending', 'held'].includes(entry.state))

try {
  await waitFor(() => rows().length === 50 && settled())
  assert.equal(rows().filter(row => row.dataset.fixtureRead === 'false').length, 25)
  button('Unread').click()
  await waitFor(() => rows().length === 50 && rows().every(row => row.dataset.fixtureRead === 'false') && settled())
  assert.match(window.notificationFixture.state.log.at(-1).url, /filter-read_eq=false/)
  button('Read').click()
  await waitFor(() => rows().length === 50 && rows().every(row => row.dataset.fixtureRead === 'true') && settled())
  assert.match(window.notificationFixture.state.log.at(-1).url, /filter-read_eq=true/)

  button('Unread').click()
  await waitFor(() => rows().every(row => row.dataset.fixtureRead === 'false'))
  const markedId = rows()[0].dataset.fixtureId
  rows()[0].querySelector('button').click()
  await waitFor(() => window.notificationFixture.state.rows.filter(row => !row.read).length === 62 && settled())
  assert.ok(!rows().some(row => row.dataset.fixtureId === markedId))

  button('Read').click()
  await waitFor(() => rows().every(row => row.dataset.fixtureRead === 'true') && settled())
  const markAll = [...window.document.querySelectorAll('button')].find(element => element.textContent.includes('Mark all as read'))
  markAll.click()
  await waitFor(() => window.notificationFixture.state.rows.every(row => row.read) && settled())
  button('Unread').click()
  await waitFor(() => rows().length === 0 && settled())
  assert.match(window.document.querySelector('.fixture-subject').textContent, /No matching notifications/)

  button('Add live unread').click()
  await waitFor(() => rows().length === 1 && settled())
  assert.equal(rows()[0].dataset.fixtureRead, 'false')
  window.notificationFixture.state.failNextWrite = true
  rows()[0].querySelector('button').click()
  await waitFor(() => window.notificationFixture.state.toasts.length === 1 && settled())
  assert.equal(rows().length, 1)
  assert.equal(rows()[0].dataset.fixtureRead, 'false')
  assert.equal(window.document.querySelectorAll('.fixture-error').length, 0)
  console.log('PASS: built-bundle synthetic DOM smoke; initial/filter URLs, mark-one, mark-all, filtered-empty, live callback, rollback. No browser, layout/pagination geometry, or backend validated.')
} finally {
  await window.happyDOM.close()
}
