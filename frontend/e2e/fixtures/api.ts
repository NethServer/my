//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Direct backend access for setup and teardown.
 *
 * Specs assert through the browser; arranging and cleaning up state is faster
 * and far more reliable over the API. The token comes from `apitool`, which runs
 * the real Logto login and exchange, so it carries real permissions rather than
 * anything signed locally.
 */

import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { registryConfig } from './personas'

const run = promisify(execFile)

const BACKEND_DIR = fileURLToPath(new URL('../../../backend', import.meta.url))

/** Base API URL, taken from the same registry the personas come from. */
export const API_URL = registryConfig.backend_url

let cachedToken: string | undefined
let minting: Promise<string> | undefined

async function mint(): Promise<string> {
  try {
    // apitool prints progress first and the token last. Spawned asynchronously:
    // the synchronous variant would block the worker's event loop for as long as
    // the sign-in takes, stalling Playwright's own protocol traffic and timers.
    const { stdout } = await run('./apitool', ['token', 'owner'], {
      cwd: BACKEND_DIR,
      encoding: 'utf8',
      timeout: 120_000,
    })
    const token = stdout.trim().split('\n').pop()?.trim()

    if (!token || token.split('.').length !== 3) {
      throw new Error(`apitool did not print a JWT, got: ${stdout.slice(-200)}`)
    }
    return token
  } catch (cause) {
    throw new Error(
      'Cannot mint an owner token. Is the backend running on :8080 and the fixture provisioned?\n' +
        '  cd backend && make run\n' +
        '  cd backend && ./apitool authz provision',
      { cause },
    )
  }
}

/**
 * A JWT for the owner. Minting one is a real sign-in, so it is cached — but the
 * access token is short-lived (`stores/login.ts`: a 20-minute refresh interval
 * with a one-minute margin) while a run may last longer, so `request` drops the
 * cache and re-mints on the first 401 rather than failing teardown. Concurrent
 * callers share one in-flight sign-in.
 */
export async function ownerToken(): Promise<string> {
  if (cachedToken) {
    return cachedToken
  }

  if (!minting) {
    minting = mint()
    minting.then(
      (token) => {
        cachedToken = token
        minting = undefined
      },
      () => {
        minting = undefined
      },
    )
  }
  return minting
}

async function send(method: string, path: string, body: unknown, token: string): Promise<Response> {
  return fetch(`${API_URL}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}

async function request(method: string, path: string, body?: unknown): Promise<Response> {
  const res = await send(method, path, body, await ownerToken())

  if (res.status !== 401) {
    return res
  }

  // The cached token outlived the backend's access-token lifetime. Re-mint once
  // and replay: a teardown that gives up here leaves the fixture behind in the
  // tenant, which is worse than the extra sign-in.
  cachedToken = undefined
  return send(method, path, body, await ownerToken())
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await request('GET', path)

  if (!res.ok) {
    throw new Error(`GET ${path} failed: ${res.status} ${await res.text()}`)
  }
  const json = (await res.json()) as { data: T }
  return json.data
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const res = await request('POST', path, body)

  if (!res.ok) {
    throw new Error(`POST ${path} failed: ${res.status} ${await res.text()}`)
  }
  const json = (await res.json()) as { data: T }
  return json.data
}

/**
 * POST with no Authorization header, for the handful of public endpoints —
 * `POST /systems/register` is the one this suite uses, and sending a token
 * would not exercise the path an appliance actually takes.
 */
export async function apiPostPublic<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    throw new Error(`POST ${path} (public) failed: ${res.status} ${await res.text()}`)
  }
  const json = (await res.json()) as { data: T }
  return json.data
}

/** Returns the HTTP status rather than throwing: teardown tolerates a 404. */
export async function apiDelete(path: string): Promise<number> {
  return (await request('DELETE', path)).status
}
