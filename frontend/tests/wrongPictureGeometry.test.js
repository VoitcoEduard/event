import test from 'node:test'
import assert from 'node:assert/strict'
import { getTargetZoom, getOverviewSize } from '../src/utils/wrongPictureGeometry.js'

test('smaller targets receive more magnification using percentage-point radii', () => {
  assert.equal(getTargetZoom(5), 7)
  assert.equal(getTargetZoom(10), 3.5)
  assert.equal(getTargetZoom(1), 12)
  assert.equal(getTargetZoom(50), 1.5)
})

test('target longest dimension occupies 70% of the crop across image orientations', () => {
  for (const aspect of [16 / 9, 1, 9 / 16]) {
    const zoom = getTargetZoom(5, aspect)
    const targetWidth = 0.1 * zoom
    const targetHeight = targetWidth / aspect
    assert.ok(Math.abs(Math.max(targetWidth, targetHeight) - 0.7) < 1e-10)
  }
})

test('invalid radii and image dimensions use safe defaults', () => {
  for (const radius of [undefined, null, NaN, Infinity, -5, 0]) {
    assert.equal(getTargetZoom(radius), 7)
  }
  assert.equal(getTargetZoom('5'), 7)
  for (const aspect of [NaN, Infinity, 0, -1]) {
    assert.equal(getTargetZoom(5, aspect), 7)
    assert.deepEqual(getOverviewSize(aspect), { width: '100%', height: '100%' })
  }
})

test('overview layer matches contained image bounds including off-center targets', () => {
  for (const [aspect, expectedWidth, expectedHeight] of [[2, 800, 400], [0.5, 400, 800], [1, 800, 800]]) {
    const size = getOverviewSize(aspect)
    const width = parseFloat(size.width) / 100 * 800
    const height = parseFloat(size.height) / 100 * 800
    assert.equal(width, expectedWidth)
    assert.equal(height, expectedHeight)
    for (const percent of [0, 10, 50, 90, 100]) {
      const imageX = (800 - expectedWidth) / 2 + expectedWidth * percent / 100
      const imageY = (800 - expectedHeight) / 2 + expectedHeight * percent / 100
      assert.equal((800 - width) / 2 + width * percent / 100, imageX)
      assert.equal((800 - height) / 2 + height * percent / 100, imageY)
    }
  }
})
