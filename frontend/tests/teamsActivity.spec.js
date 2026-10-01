import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import en from '../src/locales/en.json'
import TeamsActivityAdmin from '../src/components/TeamsActivityAdmin.vue'
import TeamsPlayerScreen from '../src/views/TeamsPlayerScreen.vue'
import TeamsConfigFields from '../src/components/TeamsConfigFields.vue'
const harness = vi.hoisted(() => ({ data: null, action: vi.fn(), join: vi.fn(), choose: vi.fn() }))
vi.mock('../src/composables/useTeamsActivity', () => ({ useTeamsActivity: () => harness.data }))
vi.mock('../src/services/teamsActivity', () => ({ teamsActivity: { action: (...args) => harness.action(...args), join: (...args) => harness.join(...args), choose: (...args) => harness.choose(...args) } }))
const module = { id: 'relay', name: 'Relay', type: 'teams', data: {} }
function activity(status = 'live') { return { id: 'a', status, config: { method: 'choice', sourceModuleId: '', teamCount: 0, targetSize: 0, capacity: 0, presets: [20] }, teams: [{ id: 'blue', name: 'Blue', color: '#3B82F6', capacity: 0 }], entrants: [{ participantId: 'anna', name: 'Anna', teamId: 'blue' }], awards: [], sourceMembers: [] } }
let wrapper
beforeEach(() => {
  vi.clearAllMocks()
  harness.data = { state: ref(null), loading: ref(false), disconnected: ref(false), error: ref(''), load: vi.fn().mockResolvedValue(undefined) }
  harness.action.mockResolvedValue({})
  const storage = new Map([['userId', 'device'], ['userName', 'Anna']])
  vi.stubGlobal('localStorage', { getItem: key => storage.get(key) || null, setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) })
})
afterEach(() => { wrapper?.unmount(); wrapper = null; vi.unstubAllGlobals() })
async function plugins() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/teams/:eventId/:moduleId/player', component: TeamsPlayerScreen }, { path: '/event', name: 'event', component: { template: '<div />' } }] })
  await router.push('/teams/event/relay/player'); await router.isReady()
  return [router, createI18n({ legacy: false, locale: 'en', messages: { en } })]
}
describe('Teams interfaces', () => {
  it('admin displays recipients and retains the request ID after ambiguous failure', async () => {
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    harness.data.state.value = { activity: activity(), participants: [], legacyTeams: [] }
    await nextTick()
    expect(wrapper.text()).toContain('This award gives points to 1 members.')
    harness.action.mockRejectedValueOnce(new TypeError('Network interrupted')).mockResolvedValueOnce({})
    await wrapper.findAll('button').find(button => button.text() === '+20').trigger('click'); await flushPromises()
    const requestId = harness.action.mock.calls[0][2].requestId
    expect(wrapper.text()).toContain('Network interrupted')
    await wrapper.findAll('button').find(button => button.text() === '+20').trigger('click'); await flushPromises()
    expect(harness.action.mock.calls[1][2].requestId).toBe(requestId)
  })
  it('player shows personal totals and animates only new awards', async () => {
    wrapper = mount(TeamsPlayerScreen, { global: { plugins: await plugins() } })
    const state = { initialized: true, name: 'Relay', status: 'live', joined: true, myTeamId: 'blue', myPoints: 20, totalPoints: 27, teams: [{ id: 'blue', name: 'Blue', color: '#3B82F6', score: 20, members: [{ name: 'Anna' }] }], awards: [{ id: 'old', points: 20, forYou: true, reason: 'Relay win' }] }
    harness.data.state.value = state; await nextTick()
    expect(wrapper.text()).toContain('27'); expect(wrapper.find('.teams-celebration').exists()).toBe(false)
    harness.data.state.value = { ...state, myPoints: 35, totalPoints: 42, awards: [...state.awards, { id: 'new', points: 15, forYou: true, reason: 'Bonus' }] }
    await nextTick(); expect(wrapper.find('.teams-celebration').text()).toContain('+15')
    expect(harness.join).not.toHaveBeenCalled()
  })
  it('completed players cannot choose or leave a team', async () => {
    wrapper = mount(TeamsPlayerScreen, { global: { plugins: await plugins() } })
    harness.data.state.value = { initialized: true, status: 'completed', joined: true, method: 'choice', myPoints: 20, totalPoints: 20, teams: [], awards: [] }
    await nextTick(); expect(wrapper.find('.player-choice').exists()).toBe(false)
    expect(wrapper.findAll('button').some(button => button.text() === 'Leave this team')).toBe(false)
  })
  it('configuration makes team count and target size mutually exclusive', async () => {
    wrapper = mount(TeamsConfigFields, { props: { modelValue: { sourceModuleId: '', method: 'random', teamCount: 0, targetSize: 4, capacity: 0, presets: [1, 5, 10] } }, global: { plugins: await plugins() } })
    await wrapper.findAll('input[type=number]')[0].setValue('3')
    expect(wrapper.emitted('update:modelValue')[0][0]).toMatchObject({ teamCount: 3, targetSize: 0 })
  })
})
