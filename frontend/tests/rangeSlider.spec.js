import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import RangeComponent from '../src/components/RangeComponent.vue'

describe('RangeComponent value clamping (Bug #8)', () => {
  it('never emits a value above max or below min when (max - min) is not divisible by step', async () => {
    const wrapper = mount(RangeComponent, {
      props: {
        min: 0,
        max: 10,
        step: 4,
        modelValue: 8,
        decimalPlaces: 0,
      },
      global: {
        stubs: { AppIcon: true },
      },
    })

    await wrapper.find('.scroll-zone').trigger('wheel', { deltaY: -100 })

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted[0][0]).toBe(10)
    expect(emitted[0][0]).toBeLessThanOrEqual(10)

    await wrapper.setProps({ modelValue: 1 })
    await wrapper.find('.scroll-zone').trigger('wheel', { deltaY: 100 })
    const emittedMin = wrapper.emitted('update:modelValue')
    expect(emittedMin[emittedMin.length - 1][0]).toBe(0)
    expect(emittedMin[emittedMin.length - 1][0]).toBeGreaterThanOrEqual(0)
  })
})
