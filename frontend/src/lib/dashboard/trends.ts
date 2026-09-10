//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '@/lib/config'
import { useLoginStore } from '@/stores/login'

export const RESOURCE_TREND_KEY = 'resourceTrend'

// Every /trend endpoint shares one contract, so one function serves all of
// them. The resource is a closed set rather than a string: a typo would answer
// 404 at runtime instead of failing to compile.
export const TREND_RESOURCES = [
  'alerts',
  'systems',
  'users',
  'applications',
  'customers',
  'resellers',
  'distributors',
] as const

export type TrendResource = (typeof TREND_RESOURCES)[number]

// The backend rejects anything else with 400, so the period is a union rather
// than a number.
export const TREND_PERIODS = [7, 30, 180, 365] as const
export type TrendPeriod = (typeof TREND_PERIODS)[number]

export interface TrendDataPoint {
  date: string
  count: number
}

export interface ResourceTrend {
  period: number
  period_label: string
  current_total: number
  previous_total: number
  delta: number
  delta_percentage: number
  trend: 'up' | 'down' | 'stable'
  data_points: TrendDataPoint[]
}

interface ResourceTrendResponse {
  code: number
  message: string
  data: ResourceTrend
}

export const getResourceTrend = (resource: TrendResource, period: TrendPeriod = 30) => {
  const loginStore = useLoginStore()

  return axios
    .get<ResourceTrendResponse>(`${API_URL}/${resource}/trend?period=${period}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export interface SparklineGeometry {
  /** `points` for a polyline: the series itself. */
  line: string
  /** `points` for a polygon: the same series closed along the baseline. */
  area: string
}

const SPARKLINE_WIDTH = 100
const SPARKLINE_HEIGHT = 32
// Keeps the stroke inside the box at both extremes instead of clipping it.
const SPARKLINE_PADDING = 2

/**
 * Maps a series into the sparkline's viewBox.
 *
 * Pure and unit-tested: a flat series draws through the middle rather than
 * dividing by a zero range, and one point draws a short horizontal line rather
 * than a single invisible coordinate.
 */
export const buildSparkline = (points: TrendDataPoint[]): SparklineGeometry => {
  if (points.length === 0) {
    return { line: '', area: '' }
  }

  const values = points.map((point) => point.count)
  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min

  const usableHeight = SPARKLINE_HEIGHT - SPARKLINE_PADDING * 2
  const step = points.length > 1 ? SPARKLINE_WIDTH / (points.length - 1) : SPARKLINE_WIDTH

  const coordinates = values.map((value, index) => {
    const x = points.length > 1 ? index * step : 0
    const ratio = range === 0 ? 0.5 : (value - min) / range
    const y = SPARKLINE_HEIGHT - SPARKLINE_PADDING - ratio * usableHeight
    return { x, y }
  })

  if (coordinates.length === 1) {
    const only = coordinates[0]
    const line = `0,${only.y} ${SPARKLINE_WIDTH},${only.y}`
    return {
      line,
      area: `0,${SPARKLINE_HEIGHT} ${line} ${SPARKLINE_WIDTH},${SPARKLINE_HEIGHT}`,
    }
  }

  const line = coordinates.map(({ x, y }) => `${round(x)},${round(y)}`).join(' ')

  return {
    line,
    area: `0,${SPARKLINE_HEIGHT} ${line} ${SPARKLINE_WIDTH},${SPARKLINE_HEIGHT}`,
  }
}

const round = (value: number) => Math.round(value * 100) / 100
