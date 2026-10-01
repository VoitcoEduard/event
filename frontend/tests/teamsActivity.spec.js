import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import en from '../src/locales/en.json'
import TeamsActivityAdmin from '../src/components/TeamsActivityAdmin.vue'
import TeamsAdminScreen from '../src/views/TeamsAdminScreen.vue'
import TeamsPlayerScreen from '../src/views/TeamsPlayerScreen.vue'
import TeamsConfigFields from '../src/components/TeamsConfigFields.vue'
const harness = vi.hoisted(() => ({ data: null, action: vi.fn(), setVisibility: vi.fn(), join: vi.fn(), choose: vi.fn() }))
vi.mock('../src/composables/useTeamsActivity', () => ({ useTeamsActivity: () => harness.data }))
vi.mock('../src/services/teamsActivity', () => ({ teamsActivity: { setVisibility: (...args) => harness.setVisibility(...args), action: (...args) => harness.action(...args), join: (...args) => harness.join(...args), choose: (...args) => harness.choose(...args) } }))
const module = { id: 'relay', name: 'Relay', type: 'teams', isActive: true, data: {} }
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
    expect(wrapper.text()).toContain('This award gives points to 1 member.')
    harness.action.mockRejectedValueOnce(new TypeError('Network interrupted')).mockResolvedValueOnce({})
    await wrapper.findAll('button').find(button => button.text() === '+20').trigger('click'); await flushPromises()
    const requestId = harness.action.mock.calls[0][2].requestId
    expect(wrapper.text()).toContain('Network interrupted')
    await wrapper.findAll('button').find(button => button.text() === '+20').trigger('click'); await flushPromises()
    expect(harness.action.mock.calls[1][2].requestId).toBe(requestId)
  })
  it('shows visibility separately from lifecycle and saves the toggle', async () => {
    harness.data.state.value = { activity: activity(), module, participants: [] }
    harness.data.load.mockImplementation(async () => { harness.data.state.value = { ...harness.data.state.value, module: { ...module, isActive: false } } })
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.text()).toContain('1 participant')
    await wrapper.find('[role="switch"]').trigger('click'); await flushPromises()
    expect(harness.setVisibility).toHaveBeenCalledWith('event', 'relay', false)
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('false')
    expect(wrapper.text()).toContain('Hidden from player home')
    expect(wrapper.find('.teams-status').text()).toBe('Live')
    expect(harness.action).not.toHaveBeenCalled()
  })
  it('keeps visibility unchanged and reports a failed toggle', async () => {
    harness.data.state.value = { activity: activity(), module, participants: [] }
    harness.setVisibility.mockRejectedValueOnce(new Error('Visibility could not be saved'))
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    await wrapper.find('[role="switch"]').trigger('click'); await flushPromises()
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.find('[role="alert"]').text()).toBe('Visibility could not be saved')
    expect(wrapper.find('[role="switch"]').attributes('disabled')).toBeUndefined()
  })
  it('requires confirmation to reopen and hides completed award controls', async () => {
    const completed = activity('completed')
    completed.awards = [{ id: 'award', teamId: 'blue', points: 20, recipients: ['anna'], createdAt: '2026-10-01T09:00:00Z' }]
    harness.data.state.value = { activity: completed, module: { ...module, isActive: false }, participants: [] }
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false).mockReturnValueOnce(true)
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    expect(wrapper.find('.award-history').text()).toContain('Reopen activity to edit awards')
    expect(wrapper.findAll('button').some(button => button.text() === 'Undo award')).toBe(false)
    expect(wrapper.find('.reopen-action').classes()).toContain('primary')
    await wrapper.find('.reopen-action').trigger('click'); await flushPromises()
    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('live and visible to players'))
    expect(harness.action).not.toHaveBeenCalled()
    await wrapper.find('.reopen-action').trigger('click'); await flushPromises()
    expect(harness.action).toHaveBeenCalledWith('event', 'relay', { action: 'reopen' })
    confirm.mockRestore()
  })
  it.each(['setup', 'live', 'completed'])('has one primary header action for %s', async status => {
    harness.data.state.value = { activity: activity(status), participants: [] }
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    const primary = wrapper.findAll('.teams-header button.primary')
    expect(primary).toHaveLength(1)
    expect(primary[0].text()).toBe({ setup: 'Start activity', live: 'Manage scores', completed: 'Reopen activity' }[status])
  })
  it('limits history to the latest five entries and can expand and collapse it', async () => {
    const live = activity()
    live.awards = Array.from({ length: 8 }, (_, index) => ({ id: `award-${index}`, teamId: 'blue', points: index + 1, recipients: ['anna'], createdAt: '2026-10-01T09:00:00Z' }))
    harness.data.state.value = { activity: live, participants: [] }
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    expect(wrapper.find('.award-history h2').text()).toContain('(8)')
    expect(wrapper.findAll('.award-history li')).toHaveLength(5)
    expect(wrapper.find('.award-history li strong').text()).toContain('+8')
    await wrapper.find('.award-history > button').trigger('click')
    expect(wrapper.findAll('.award-history li')).toHaveLength(8)
    await wrapper.find('.award-history > button').trigger('click')
    expect(wrapper.findAll('.award-history li')).toHaveLength(5)
  })
  it('assigns an existing entrant from a compact empty team card', async () => {
    const live = activity()
    live.teams.push({ id: 'empty', name: 'Empty', color: '#000000' })
    live.entrants.push({ participantId: 'ben', name: 'Ben', teamId: '' })
    harness.data.state.value = { activity: live, participants: [] }
    wrapper = mount(TeamsActivityAdmin, { props: { eventId: 'event', module, modules: [module] }, global: { plugins: await plugins() } })
    const card = wrapper.findAll('.team-card').find(card => card.find('h2').text() === 'Empty')
    expect(card.find('.award-buttons').exists()).toBe(false)
    expect(card.find('.custom-award').exists()).toBe(false)
    await card.find('.team-empty button').trigger('click')
    await wrapper.find('[role="dialog"] select').setValue('ben')
    await wrapper.find('[role="dialog"]').trigger('submit'); await flushPromises()
    expect(harness.action).toHaveBeenCalledWith('event', 'relay', { action: 'assign', participantId: 'ben', teamId: 'empty' })
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })
  it.each([1, 2])('only shows the activity selector for multiple activities (%i)', async count => {
    const modules = Array.from({ length: count }, (_, index) => ({ ...module, id: index ? 'second' : module.id, name: index ? 'Second' : module.name }))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ name: 'Event', modules }) }))
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/admin/event/:id/teams', component: TeamsAdminScreen }] })
    await router.push('/admin/event/event/teams?moduleId=relay'); await router.isReady()
    wrapper = mount(TeamsAdminScreen, { global: { plugins: [router, createI18n({ legacy: false, locale: 'en', messages: { en } })], stubs: { TeamsActivityAdmin: true } } })
    await flushPromises()
    expect(wrapper.find('.teams-admin-picker').exists()).toBe(count > 1)
    expect(wrapper.find('h1').text()).toBe('Relay')
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
