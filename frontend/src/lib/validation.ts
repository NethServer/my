//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

export const isValidationErrorCode = (errorCode: number) => {
  return [400, 409, 422].includes(errorCode)
}
