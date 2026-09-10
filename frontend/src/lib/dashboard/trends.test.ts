//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { buildSparkline, type TrendDataPoint } from './trends'

const series = (...counts: number[]): TrendDataPoint[] =>
  counts.map((count, index) => ({ date: `2026-09-${String(index + 1).padStart(2, '0')}`, count }))

const coordinates = (points: string) =>
  points
    .trim()
    .split(' ')
    .filter(Boolean)
    .map((pair) => {
      const [x, y] = pair.split(',').map(Number)
      return { x, y }
    })

describe('buildSparkline', () => {
  it('returns nothing to draw for an empty series', () => {
    expect(buildSparkline([])).toEqual({ line: '', area: '' })
  })

  it('draws a flat series through the middle rather than dividing by a zero range', () => {
    const { line } = buildSparkline(series(4, 4, 4, 4))
    const points = coordinates(line)

    expect(points).toHaveLength(4)
    const [first] = points
    for (const point of points) {
      expect(point.y).toBe(first.y)
      expect(Number.isFinite(point.y)).toBe(true)
    }
    // exactly halfway down the 32-unit box, padding included
    expect(first.y).toBe(16)
  })

  it('draws a single point as a horizontal line spanning the box', () => {
    const points = coordinates(buildSparkline(series(7)).line)

    expect(points).toHaveLength(2)
    expect(points[0].x).toBe(0)
    expect(points[1].x).toBe(100)
    expect(points[0].y).toBe(points[1].y)
  })

  it('maps values monotonically, with the highest count nearest the top', () => {
    const points = coordinates(buildSparkline(series(0, 5, 10)).line)

    // SVG y grows downward, so a rising series has falling coordinates
    expect(points[0].y).toBeGreaterThan(points[1].y)
    expect(points[1].y).toBeGreaterThan(points[2].y)
  })

  it('keeps every coordinate inside the viewBox', () => {
    const points = coordinates(buildSparkline(series(3, 90, 12, 0, 45)).line)

    for (const point of points) {
      expect(point.x).toBeGreaterThanOrEqual(0)
      expect(point.x).toBeLessThanOrEqual(100)
      expect(point.y).toBeGreaterThanOrEqual(0)
      expect(point.y).toBeLessThanOrEqual(32)
    }
  })

  it('spreads the points evenly from edge to edge', () => {
    const points = coordinates(buildSparkline(series(1, 2, 3, 4, 5)).line)

    expect(points[0].x).toBe(0)
    expect(points[points.length - 1].x).toBe(100)
  })

  it('closes the area along the baseline so the fill sits under the line', () => {
    const { line, area } = buildSparkline(series(1, 2, 3))

    expect(area).toContain(line)
    expect(area.startsWith('0,32')).toBe(true)
    expect(area.endsWith('100,32')).toBe(true)
  })
})
