//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import axios, { AxiosError, AxiosHeaders, type InternalAxiosRequestConfig } from 'axios'

const push = vi.fn()
const createNotificationFromAxiosError = vi.fn()

vi.mock('@/router', () => ({ default: { push: (...args: unknown[]) => push(...args) } }))
vi.mock('@/stores/login', () => ({ useLoginStore: () => ({}) }))
vi.mock('@/stores/notifications', () => ({
  useNotificationsStore: () => ({ createNotificationFromAxiosError }),
}))

const { configureAxios } = await import('./axios')
configureAxios()

// the handler configureAxios installed, called the way axios calls it
type Handler = { rejected: (error: unknown) => Promise<unknown> }
const handlers = (axios.interceptors.response as unknown as { handlers: Handler[] }).handlers
const onRejected = handlers[handlers.length - 1]!.rejected

function forbidden(method: string, data: Record<string, unknown> = {}) {
  const config = { method, url: '/api/users', headers: new AxiosHeaders() }
  return new AxiosError(
    'Request failed with status code 403',
    'ERR_BAD_REQUEST',
    config as InternalAxiosRequestConfig,
    null,
    {
      status: 403,
      statusText: 'Forbidden',
      headers: {},
      config: config as InternalAxiosRequestConfig,
      data: { code: 403, ...data },
    },
  )
}

describe('403 response interceptor', () => {
  beforeEach(() => {
    push.mockClear()
    createNotificationFromAxiosError.mockClear()
  })

  it.each(['get', 'head'])('sends a refused %s to the forbidden page', async (method) => {
    await expect(onRejected(forbidden(method))).rejects.toBeInstanceOf(AxiosError)
    expect(push).toHaveBeenCalledWith({ name: 'forbidden' })
    expect(createNotificationFromAxiosError).not.toHaveBeenCalled()
  })

  it.each(['post', 'put', 'patch', 'delete'])(
    'keeps the page on a refused %s and shows the reason',
    async (method) => {
      const error = forbidden(method, { message: 'access denied: only the owner role can' })
      await expect(onRejected(error)).rejects.toBe(error)
      expect(push).not.toHaveBeenCalled()
      expect(error.message).toBe('access denied: only the owner role can')
      expect(createNotificationFromAxiosError).toHaveBeenCalledWith(error)
    },
  )

  it('keeps the axios message when the backend sends none', async () => {
    const error = forbidden('post')
    await expect(onRejected(error)).rejects.toBe(error)
    expect(error.message).toBe('Request failed with status code 403')
  })

  it('leaves an impersonation refusal to the caller', async () => {
    const error = forbidden('get', { data: { reason: 'impersonation_active' } })
    await expect(onRejected(error)).rejects.toBe(error)
    expect(push).not.toHaveBeenCalled()
    expect(createNotificationFromAxiosError).not.toHaveBeenCalled()
  })
})
