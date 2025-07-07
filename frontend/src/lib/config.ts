//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

export const PRODUCT_NAME = 'My Nethesis'

// path requested by the OIDC provider to redirect the user after a successful login
export const LOGIN_REDIRECT_URI = `${window.location.origin}/${import.meta.env.VITE_SIGNIN_REDIRECT_URI}`

// go to the login page after a successful sign out
export const SIGN_OUT_REDIRECT_URI = `${window.location.origin}/${import.meta.env.VITE_SIGNOUT_REDIRECT_URI}`

// logto configuration
export const LOGTO_ENDPOINT = import.meta.env.VITE_LOGTO_ENDPOINT
export const LOGTO_APP_ID = import.meta.env.VITE_LOGTO_APP_ID

export const API_URL = import.meta.env.VITE_API_BASE_URL
