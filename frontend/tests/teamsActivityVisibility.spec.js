import { afterEach, describe, expect, it, vi } from 'vitest'
import { teamsActivity } from '../src/services/teamsActivity'

vi.mock('../src/utils', () => ({ API_BASE_URL: '/api', getAuthHeaders: () => ({ 'X-CSRF-Token': 'test-csrf' }), getUserId: () => 'player' }))
afterEach(() => vi.unstubAllGlobals())

describe('Teams player visibility', () => {
  it('updates module visibility with authenticated PUT without a lifecycle action', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetch)
    await teamsActivity.setVisibility('event/one', 'relay/two', false)
    expect(fetch).toHaveBeenCalledWith('/api/events/event%2Fone/modules/relay%2Ftwo/status', {
      method: 'PUT', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': 'test-csrf' },
      body: JSON.stringify({ isActive: false })
    })
  })

  it('reports server errors so the interface can retain the previous visibility', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 403, text: async () => ' Forbidden ' }))
    await expect(teamsActivity.setVisibility('event', 'relay', true)).rejects.toThrow('Forbidden')
  })
})
