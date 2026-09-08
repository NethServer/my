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

import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { registryConfig } from './personas'

const BACKEND_DIR = fileURLToPath(new URL('../../../backend', import.meta.url))

/** Base API URL, taken from the same registry the personas come from. */
export const API_URL = registryConfig.backend_url

let cachedToken: string | undefined

/**
 * A JWT for the owner. Minting one is a real sign-in, so it is cached for the
 * lifetime of the process.
 */
export function ownerToken(): string {
  if (cachedToken) {
    return cachedToken
  }

  try {
    // apitool prints progress first and the token last.
    const out = execFileSync('./apitool', ['token', 'owner'], {
      cwd: BACKEND_DIR,
      encoding: 'utf8',
      timeout: 120_000,
    })
    const token = out.trim().split('\n').pop()?.trim()

    if (!token || token.split('.').length !== 3) {
      throw new Error(`apitool did not print a JWT, got: ${out.slice(-200)}`)
    }
    cachedToken = token
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

async function request(method: string, path: string, body?: unknown): Promise<Response> {
  return fetch(`${API_URL}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${ownerToken()}`,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
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
