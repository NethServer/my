//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest'
import axios, { AxiosError, AxiosHeaders, type InternalAxiosRequestConfig } from 'axios'

const push = vi.fn()
const createNotificationFromAxiosError = vi.fn()

vi.mock('@/i18n', () => ({ default: { global: { t: (key: string) => key } } }))
vi.mock('@/router', () => ({ default: { push: (...args: unknown[]) => push(...args) } }))
const loginStore = {
  jwtToken: '',
  doRefreshToken: vi.fn(async () => false),
  forceReauth: vi.fn(),
  shouldRefreshToken: () => false,
  pendingRefresh: async () => {},
}
vi.mock('@/stores/login', () => ({ useLoginStore: () => loginStore }))
vi.mock('@/stores/notifications', () => ({
  useNotificationsStore: () => ({ createNotificationFromAxiosError }),
}))

const { configureAxios } = await import('./axios')
const { API_URL } = await import('./config')
configureAxios()

// the handler configureAxios installed, called the way axios calls it
type Handler = { rejected: (error: unknown) => Promise<unknown> }
const handlers = (axios.interceptors.response as unknown as { handlers: Handler[] }).handlers
const onRejected = handlers[handlers.length - 1]!.rejected

function rejected(status: number, method: string, data: Record<string, unknown> = {}) {
  const config = { method, url: '/api/users', headers: new AxiosHeaders() }
  return new AxiosError(
    `Request failed with status code ${status}`,
    'ERR_BAD_REQUEST',
    config as InternalAxiosRequestConfig,
    null,
    {
      status,
      statusText: '',
      headers: {},
      config: config as InternalAxiosRequestConfig,
      data: { code: status, ...data },
    },
  )
}

const forbidden = (method: string, data: Record<string, unknown> = {}) =>
  rejected(403, method, data)

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
      expect(error.message).toBe('Access denied: only the owner role can')
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

// the forms show error.message: a conflict with no field errors reaches the
// form's notification only through it, while a server fault or a lost
// connection gets a fixed text
describe('error message', () => {
  it('carries the backend reason on any refused request', async () => {
    const error = rejected(409, 'put', { message: 'system reassignment in progress' })
    await expect(onRejected(error)).rejects.toBe(error)
    expect(error.message).toBe('System reassignment in progress')
  })

  it('replaces the message of a server fault', async () => {
    const error = rejected(500, 'get', { message: 'failed to query users: connection refused' })
    await expect(onRejected(error)).rejects.toBe(error)
    expect(error.message).toBe('common.something_went_wrong')
  })

  it('explains a request that got no answer', async () => {
    const config = { method: 'get', url: '/api/users', headers: new AxiosHeaders() }
    const error = new AxiosError(
      'Network Error',
      'ERR_NETWORK',
      config as InternalAxiosRequestConfig,
    )
    await expect(onRejected(error)).rejects.toBe(error)
    expect(error.message).toBe('common.network_error')
  })
})

describe('401 response interceptor', () => {
  // a refused request that axios replays: the adapter answers the replay
  // without a network, and records the token it carried
  function unauthorized(sentToken: string) {
    const replayedWith: string[] = []
    const config = {
      method: 'get',
      url: `${API_URL}/systems`,
      headers: new AxiosHeaders({ Authorization: `Bearer ${sentToken}` }),
      adapter: async (replay: InternalAxiosRequestConfig) => {
        replayedWith.push(String(replay.headers.Authorization))
        return { data: { ok: true }, status: 200, statusText: 'OK', headers: {}, config: replay }
      },
    }
    const error = new AxiosError(
      'Request failed with status code 401',
      'ERR_BAD_REQUEST',
      config as unknown as InternalAxiosRequestConfig,
      null,
      {
        status: 401,
        statusText: '',
        headers: {},
        config: config as unknown as InternalAxiosRequestConfig,
        data: { code: 401 },
      },
    )
    return { error, replayedWith }
  }

  beforeEach(() => {
    loginStore.jwtToken = 'current-token'
    loginStore.doRefreshToken.mockClear()
    loginStore.forceReauth.mockClear()
  })

  it('replays with the current token a request sent with a replaced one', async () => {
    const { error, replayedWith } = unauthorized('old-token')

    await expect(onRejected(error)).resolves.toMatchObject({ status: 200 })
    expect(replayedWith).toEqual(['Bearer current-token'])
    expect(loginStore.doRefreshToken).not.toHaveBeenCalled()
    expect(loginStore.forceReauth).not.toHaveBeenCalled()
  })

  it('re-authenticates when the current token itself is refused and cannot be refreshed', async () => {
    const { error, replayedWith } = unauthorized('current-token')

    await expect(onRejected(error)).rejects.toBe(error)
    expect(replayedWith).toEqual([])
    expect(loginStore.doRefreshToken).toHaveBeenCalledOnce()
    expect(loginStore.forceReauth).toHaveBeenCalledOnce()
  })
})
