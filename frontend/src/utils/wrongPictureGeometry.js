// Radius is stored in percentage points, not as a fraction.
export function getTargetZoom(radius, aspectRatio = 1) {
  const parsed = Number(radius)
  const safeRadius = Number.isFinite(parsed) && parsed > 0 ? Math.min(50, parsed) : 5
  const aspect = Number.isFinite(aspectRatio) && aspectRatio > 0 ? aspectRatio : 1
  // Fit the target's longest dimension into 70% of the square crop.
  // Keep the original zoom limits relative to the image's longest dimension.
  return Math.min(12, Math.max(1.5, 70 / (2 * safeRadius))) * Math.min(1, aspect)
}

export function getOverviewSize(aspectRatio = 1) {
  const aspect = Number.isFinite(aspectRatio) && aspectRatio > 0 ? aspectRatio : 1
  return {
    width: `${Math.min(1, aspect) * 100}%`,
    height: `${Math.min(1, 1 / aspect) * 100}%`
  }
}
