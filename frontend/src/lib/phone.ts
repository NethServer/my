//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { parsePhoneNumberFromString } from 'libphonenumber-js'
import type { NeComboboxOption } from '@nethesis/vue-components'

// The backend stores phone numbers as digits-only (E.164 without the leading
// `+`), e.g. "393330001113". This helper renders a human-readable version for
// display, e.g. "+39 333 000 1113".
//
// Inputs we need to handle:
// - "" / null / undefined → return ""
// - "393330001113"        → "+39 333 000 1113"
// - "+39 333 0001113"     → "+39 333 000 1113" (already-formatted legacy values
//                           still in the DB get re-parsed and re-formatted)
// - non-parseable values  → returned untouched as a fallback
export function formatPhoneForDisplay(raw: string | null | undefined): string {
  if (!raw) {
    return ''
  }
  // libphonenumber-js needs a leading "+" to detect the country from the digits.
  // If the raw value already has it, parse as-is; otherwise prepend.
  const candidate = raw.startsWith('+') ? raw : `+${raw}`
  const parsed = parsePhoneNumberFromString(candidate)
  return parsed?.formatInternational() ?? raw
}

// Parse a phone number and extract the country code (iso2) and local part.
// Returns { countryIso2, localPart } or null if parsing fails.
// Inputs we handle:
// - "" / null / undefined → return null
// - "393330001113"        → { countryIso2: "it", localPart: "333 000 1113" }
// - "+39 333 0001113"     → { countryIso2: "it", localPart: "333 000 1113" }
export function parsePhoneNumber(
  raw: string | null | undefined,
): { countryIso2: string; localPart: string } | null {
  if (!raw) {
    return null
  }

  // libphonenumber-js needs a leading "+" to detect the country from the digits.
  // If the raw value already has it, parse as-is; otherwise prepend.
  const candidate = raw.startsWith('+') ? raw : `+${raw}`
  const parsed = parsePhoneNumberFromString(candidate)

  if (!parsed || !parsed.country) {
    return null
  }

  // Get the country code in lowercase (e.g., "IT" → "it")
  const countryIso2 = parsed.country.toLowerCase()
  // Format the local part (e.g., "333 000 1113")
  const localPart = parsed.formatNational()

  return { countryIso2, localPart }
}

// Combine country code ISO2 and phone local part into a backend-compatible format.
// Returns the phone number as digits-only (E.164 without the leading `+`),
// e.g., "it" + "333 000 1113" → "393330001113"
// If countryCode or localPart is empty, returns empty string.
export function combinePhoneParts(countryIso2: string, localPart: string): string {
  if (!countryIso2 || !localPart) {
    return ''
  }

  // Find the country code from the countries array
  const country = countries.find((c) => c.iso2 === countryIso2)
  if (!country) {
    return ''
  }

  // Remove all non-digit characters from the local part, but keep the leading "+"
  // if present (which indicates the user incorrectly entered the country prefix).
  // This lets us detect and warn about malformed input.
  const digitsOnly = localPart.replace(/[^+\d]/g, '')

  // Concatenate country code + local part digits
  return `${country.country_code}${digitsOnly}`
}

export const countries = [
  { iso2: 'af', country_name: 'Afghanistan', country_code: '93', flag: '🇦🇫' },
  { iso2: 'ax', country_name: 'Åland Islands', country_code: '358', flag: '🇦🇽' },
  { iso2: 'al', country_name: 'Albania', country_code: '355', flag: '🇦🇱' },
  { iso2: 'dz', country_name: 'Algeria', country_code: '213', flag: '🇩🇿' },
  { iso2: 'as', country_name: 'American Samoa', country_code: '1', flag: '🇦🇸' },
  { iso2: 'ad', country_name: 'Andorra', country_code: '376', flag: '🇦🇩' },
  { iso2: 'ao', country_name: 'Angola', country_code: '244', flag: '🇦🇴' },
  { iso2: 'ai', country_name: 'Anguilla', country_code: '1', flag: '🇦🇮' },
  { iso2: 'ag', country_name: 'Antigua and Barbuda', country_code: '1', flag: '🇦🇬' },
  { iso2: 'ar', country_name: 'Argentina', country_code: '54', flag: '🇦🇷' },
  { iso2: 'am', country_name: 'Armenia', country_code: '374', flag: '🇦🇲' },
  { iso2: 'aw', country_name: 'Aruba', country_code: '297', flag: '🇦🇼' },
  { iso2: 'ac', country_name: 'Ascension Island', country_code: '247', flag: '🇦🇨' },
  { iso2: 'au', country_name: 'Australia', country_code: '61', flag: '🇦🇺' },
  { iso2: 'at', country_name: 'Austria', country_code: '43', flag: '🇦🇹' },
  { iso2: 'az', country_name: 'Azerbaijan', country_code: '994', flag: '🇦🇿' },
  { iso2: 'bs', country_name: 'Bahamas', country_code: '1', flag: '🇧🇸' },
  { iso2: 'bh', country_name: 'Bahrain', country_code: '973', flag: '🇧🇭' },
  { iso2: 'bd', country_name: 'Bangladesh', country_code: '880', flag: '🇧🇩' },
  { iso2: 'bb', country_name: 'Barbados', country_code: '1', flag: '🇧🇧' },
  { iso2: 'by', country_name: 'Belarus', country_code: '375', flag: '🇧🇾' },
  { iso2: 'be', country_name: 'Belgium', country_code: '32', flag: '🇧🇪' },
  { iso2: 'bz', country_name: 'Belize', country_code: '501', flag: '🇧🇿' },
  { iso2: 'bj', country_name: 'Benin', country_code: '229', flag: '🇧🇯' },
  { iso2: 'bm', country_name: 'Bermuda', country_code: '1', flag: '🇧🇲' },
  { iso2: 'bt', country_name: 'Bhutan', country_code: '975', flag: '🇧🇹' },
  { iso2: 'bo', country_name: 'Bolivia', country_code: '591', flag: '🇧🇴' },
  { iso2: 'ba', country_name: 'Bosnia and Herzegovina', country_code: '387', flag: '🇧🇦' },
  { iso2: 'bw', country_name: 'Botswana', country_code: '267', flag: '🇧🇼' },
  { iso2: 'br', country_name: 'Brazil', country_code: '55', flag: '🇧🇷' },
  { iso2: 'io', country_name: 'British Indian Ocean Territory', country_code: '246', flag: '🇮🇴' },
  { iso2: 'vg', country_name: 'British Virgin Islands', country_code: '1', flag: '🇻🇬' },
  { iso2: 'bn', country_name: 'Brunei', country_code: '673', flag: '🇧🇳' },
  { iso2: 'bg', country_name: 'Bulgaria', country_code: '359', flag: '🇧🇬' },
  { iso2: 'bf', country_name: 'Burkina Faso', country_code: '226', flag: '🇧🇫' },
  { iso2: 'bi', country_name: 'Burundi', country_code: '257', flag: '🇧🇮' },
  { iso2: 'kh', country_name: 'Cambodia', country_code: '855', flag: '🇰🇭' },
  { iso2: 'cm', country_name: 'Cameroon', country_code: '237', flag: '🇨🇲' },
  { iso2: 'ca', country_name: 'Canada', country_code: '1', flag: '🇨🇦' },
  { iso2: 'cv', country_name: 'Cape Verde', country_code: '238', flag: '🇨🇻' },
  { iso2: 'bq', country_name: 'Caribbean Netherlands', country_code: '599', flag: '🇧🇶' },
  { iso2: 'ky', country_name: 'Cayman Islands', country_code: '1', flag: '🇰🇾' },
  { iso2: 'cf', country_name: 'Central African Republic', country_code: '236', flag: '🇨🇫' },
  { iso2: 'td', country_name: 'Chad', country_code: '235', flag: '🇹🇩' },
  { iso2: 'cl', country_name: 'Chile', country_code: '56', flag: '🇨🇱' },
  { iso2: 'cn', country_name: 'China', country_code: '86', flag: '🇨🇳' },
  { iso2: 'cx', country_name: 'Christmas Island', country_code: '61', flag: '🇨🇽' },
  { iso2: 'cc', country_name: 'Cocos (Keeling) Islands', country_code: '61', flag: '🇨🇨' },
  { iso2: 'co', country_name: 'Colombia', country_code: '57', flag: '🇨🇴' },
  { iso2: 'km', country_name: 'Comoros', country_code: '269', flag: '🇰🇲' },
  { iso2: 'cg', country_name: 'Congo (Brazzaville)', country_code: '242', flag: '🇨🇬' },
  { iso2: 'cd', country_name: 'Congo (Kinshasa)', country_code: '243', flag: '🇨🇩' },
  { iso2: 'ck', country_name: 'Cook Islands', country_code: '682', flag: '🇨🇰' },
  { iso2: 'cr', country_name: 'Costa Rica', country_code: '506', flag: '🇨🇷' },
  { iso2: 'ci', country_name: "Côte d'Ivoire", country_code: '225', flag: '🇨🇮' },
  { iso2: 'hr', country_name: 'Croatia', country_code: '385', flag: '🇭🇷' },
  { iso2: 'cu', country_name: 'Cuba', country_code: '53', flag: '🇨🇺' },
  { iso2: 'cw', country_name: 'Curaçao', country_code: '599', flag: '🇨🇼' },
  { iso2: 'cy', country_name: 'Cyprus', country_code: '357', flag: '🇨🇾' },
  { iso2: 'cz', country_name: 'Czech Republic', country_code: '420', flag: '🇨🇿' },
  { iso2: 'dk', country_name: 'Denmark', country_code: '45', flag: '🇩🇰' },
  { iso2: 'dj', country_name: 'Djibouti', country_code: '253', flag: '🇩🇯' },
  { iso2: 'dm', country_name: 'Dominica', country_code: '1', flag: '🇩🇲' },
  { iso2: 'do', country_name: 'Dominican Republic', country_code: '1', flag: '🇩🇴' },
  { iso2: 'ec', country_name: 'Ecuador', country_code: '593', flag: '🇪🇨' },
  { iso2: 'eg', country_name: 'Egypt', country_code: '20', flag: '🇪🇬' },
  { iso2: 'sv', country_name: 'El Salvador', country_code: '503', flag: '🇸🇻' },
  { iso2: 'gq', country_name: 'Equatorial Guinea', country_code: '240', flag: '🇬🇶' },
  { iso2: 'er', country_name: 'Eritrea', country_code: '291', flag: '🇪🇷' },
  { iso2: 'ee', country_name: 'Estonia', country_code: '372', flag: '🇪🇪' },
  { iso2: 'sz', country_name: 'Eswatini', country_code: '268', flag: '🇸🇿' },
  { iso2: 'et', country_name: 'Ethiopia', country_code: '251', flag: '🇪🇹' },
  { iso2: 'fk', country_name: 'Falkland Islands (Malvinas)', country_code: '500', flag: '🇫🇰' },
  { iso2: 'fo', country_name: 'Faroe Islands', country_code: '298', flag: '🇫🇴' },
  { iso2: 'fj', country_name: 'Fiji', country_code: '679', flag: '🇫🇯' },
  { iso2: 'fi', country_name: 'Finland', country_code: '358', flag: '🇫🇮' },
  { iso2: 'fr', country_name: 'France', country_code: '33', flag: '🇫🇷' },
  { iso2: 'gf', country_name: 'French Guiana', country_code: '594', flag: '🇬🇫' },
  { iso2: 'pf', country_name: 'French Polynesia', country_code: '689', flag: '🇵🇫' },
  { iso2: 'ga', country_name: 'Gabon', country_code: '241', flag: '🇬🇦' },
  { iso2: 'gm', country_name: 'Gambia', country_code: '220', flag: '🇬🇲' },
  { iso2: 'ge', country_name: 'Georgia', country_code: '995', flag: '🇬🇪' },
  { iso2: 'de', country_name: 'Germany', country_code: '49', flag: '🇩🇪' },
  { iso2: 'gh', country_name: 'Ghana', country_code: '233', flag: '🇬🇭' },
  { iso2: 'gi', country_name: 'Gibraltar', country_code: '350', flag: '🇬🇮' },
  { iso2: 'gr', country_name: 'Greece', country_code: '30', flag: '🇬🇷' },
  { iso2: 'gl', country_name: 'Greenland', country_code: '299', flag: '🇬🇱' },
  { iso2: 'gd', country_name: 'Grenada', country_code: '1', flag: '🇬🇩' },
  { iso2: 'gp', country_name: 'Guadeloupe', country_code: '590', flag: '🇬🇵' },
  { iso2: 'gu', country_name: 'Guam', country_code: '1', flag: '🇬🇺' },
  { iso2: 'gt', country_name: 'Guatemala', country_code: '502', flag: '🇬🇹' },
  { iso2: 'gg', country_name: 'Guernsey', country_code: '44', flag: '🇬🇬' },
  { iso2: 'gn', country_name: 'Guinea', country_code: '224', flag: '🇬🇳' },
  { iso2: 'gw', country_name: 'Guinea-Bissau', country_code: '245', flag: '🇬🇼' },
  { iso2: 'gy', country_name: 'Guyana', country_code: '592', flag: '🇬🇾' },
  { iso2: 'ht', country_name: 'Haiti', country_code: '509', flag: '🇭🇹' },
  { iso2: 'hn', country_name: 'Honduras', country_code: '504', flag: '🇭🇳' },
  { iso2: 'hk', country_name: 'Hong Kong SAR China', country_code: '852', flag: '🇭🇰' },
  { iso2: 'hu', country_name: 'Hungary', country_code: '36', flag: '🇭🇺' },
  { iso2: 'is', country_name: 'Iceland', country_code: '354', flag: '🇮🇸' },
  { iso2: 'in', country_name: 'India', country_code: '91', flag: '🇮🇳' },
  { iso2: 'id', country_name: 'Indonesia', country_code: '62', flag: '🇮🇩' },
  { iso2: 'ir', country_name: 'Iran', country_code: '98', flag: '🇮🇷' },
  { iso2: 'iq', country_name: 'Iraq', country_code: '964', flag: '🇮🇶' },
  { iso2: 'ie', country_name: 'Ireland', country_code: '353', flag: '🇮🇪' },
  { iso2: 'im', country_name: 'Isle of Man', country_code: '44', flag: '🇮🇲' },
  { iso2: 'il', country_name: 'Israel', country_code: '972', flag: '🇮🇱' },
  { iso2: 'it', country_name: 'Italy', country_code: '39', flag: '🇮🇹' },
  { iso2: 'jm', country_name: 'Jamaica', country_code: '1', flag: '🇯🇲' },
  { iso2: 'jp', country_name: 'Japan', country_code: '81', flag: '🇯🇵' },
  { iso2: 'je', country_name: 'Jersey', country_code: '44', flag: '🇯🇪' },
  { iso2: 'jo', country_name: 'Jordan', country_code: '962', flag: '🇯🇴' },
  { iso2: 'kz', country_name: 'Kazakhstan', country_code: '7', flag: '🇰🇿' },
  { iso2: 'ke', country_name: 'Kenya', country_code: '254', flag: '🇰🇪' },
  { iso2: 'ki', country_name: 'Kiribati', country_code: '686', flag: '🇰🇮' },
  { iso2: 'xk', country_name: 'Kosovo', country_code: '383', flag: '🇽🇰' },
  { iso2: 'kw', country_name: 'Kuwait', country_code: '965', flag: '🇰🇼' },
  { iso2: 'kg', country_name: 'Kyrgyzstan', country_code: '996', flag: '🇰🇬' },
  { iso2: 'la', country_name: 'Laos', country_code: '856', flag: '🇱🇦' },
  { iso2: 'lv', country_name: 'Latvia', country_code: '371', flag: '🇱🇻' },
  { iso2: 'lb', country_name: 'Lebanon', country_code: '961', flag: '🇱🇧' },
  { iso2: 'ls', country_name: 'Lesotho', country_code: '266', flag: '🇱🇸' },
  { iso2: 'lr', country_name: 'Liberia', country_code: '231', flag: '🇱🇷' },
  { iso2: 'ly', country_name: 'Libya', country_code: '218', flag: '🇱🇾' },
  { iso2: 'li', country_name: 'Liechtenstein', country_code: '423', flag: '🇱🇮' },
  { iso2: 'lt', country_name: 'Lithuania', country_code: '370', flag: '🇱🇹' },
  { iso2: 'lu', country_name: 'Luxembourg', country_code: '352', flag: '🇱🇺' },
  { iso2: 'mo', country_name: 'Macao SAR China', country_code: '853', flag: '🇲🇴' },
  { iso2: 'mg', country_name: 'Madagascar', country_code: '261', flag: '🇲🇬' },
  { iso2: 'mw', country_name: 'Malawi', country_code: '265', flag: '🇲🇼' },
  { iso2: 'my', country_name: 'Malaysia', country_code: '60', flag: '🇲🇾' },
  { iso2: 'mv', country_name: 'Maldives', country_code: '960', flag: '🇲🇻' },
  { iso2: 'ml', country_name: 'Mali', country_code: '223', flag: '🇲🇱' },
  { iso2: 'mt', country_name: 'Malta', country_code: '356', flag: '🇲🇹' },
  { iso2: 'mh', country_name: 'Marshall Islands', country_code: '692', flag: '🇲🇭' },
  { iso2: 'mq', country_name: 'Martinique', country_code: '596', flag: '🇲🇶' },
  { iso2: 'mr', country_name: 'Mauritania', country_code: '222', flag: '🇲🇷' },
  { iso2: 'mu', country_name: 'Mauritius', country_code: '230', flag: '🇲🇺' },
  { iso2: 'yt', country_name: 'Mayotte', country_code: '262', flag: '🇾🇹' },
  { iso2: 'mx', country_name: 'Mexico', country_code: '52', flag: '🇲🇽' },
  { iso2: 'fm', country_name: 'Micronesia', country_code: '691', flag: '🇫🇲' },
  { iso2: 'md', country_name: 'Moldova', country_code: '373', flag: '🇲🇩' },
  { iso2: 'mc', country_name: 'Monaco', country_code: '377', flag: '🇲🇨' },
  { iso2: 'mn', country_name: 'Mongolia', country_code: '976', flag: '🇲🇳' },
  { iso2: 'me', country_name: 'Montenegro', country_code: '382', flag: '🇲🇪' },
  { iso2: 'ms', country_name: 'Montserrat', country_code: '1', flag: '🇲🇸' },
  { iso2: 'ma', country_name: 'Morocco', country_code: '212', flag: '🇲🇦' },
  { iso2: 'mz', country_name: 'Mozambique', country_code: '258', flag: '🇲🇿' },
  { iso2: 'mm', country_name: 'Myanmar (Burma)', country_code: '95', flag: '🇲🇲' },
  { iso2: 'na', country_name: 'Namibia', country_code: '264', flag: '🇳🇦' },
  { iso2: 'nr', country_name: 'Nauru', country_code: '674', flag: '🇳🇷' },
  { iso2: 'np', country_name: 'Nepal', country_code: '977', flag: '🇳🇵' },
  { iso2: 'nl', country_name: 'Netherlands', country_code: '31', flag: '🇳🇱' },
  { iso2: 'nc', country_name: 'New Caledonia', country_code: '687', flag: '🇳🇨' },
  { iso2: 'nz', country_name: 'New Zealand', country_code: '64', flag: '🇳🇿' },
  { iso2: 'ni', country_name: 'Nicaragua', country_code: '505', flag: '🇳🇮' },
  { iso2: 'ne', country_name: 'Niger', country_code: '227', flag: '🇳🇪' },
  { iso2: 'ng', country_name: 'Nigeria', country_code: '234', flag: '🇳🇬' },
  { iso2: 'nu', country_name: 'Niue', country_code: '683', flag: '🇳🇺' },
  { iso2: 'nf', country_name: 'Norfolk Island', country_code: '672', flag: '🇳🇫' },
  { iso2: 'kp', country_name: 'North Korea', country_code: '850', flag: '🇰🇵' },
  { iso2: 'mk', country_name: 'North Macedonia', country_code: '389', flag: '🇲🇰' },
  { iso2: 'mp', country_name: 'Northern Mariana Islands', country_code: '1', flag: '🇲🇵' },
  { iso2: 'no', country_name: 'Norway', country_code: '47', flag: '🇳🇴' },
  { iso2: 'om', country_name: 'Oman', country_code: '968', flag: '🇴🇲' },
  { iso2: 'pk', country_name: 'Pakistan', country_code: '92', flag: '🇵🇰' },
  { iso2: 'pw', country_name: 'Palau', country_code: '680', flag: '🇵🇼' },
  { iso2: 'ps', country_name: 'Palestinian Territories', country_code: '970', flag: '🇵🇸' },
  { iso2: 'pa', country_name: 'Panama', country_code: '507', flag: '🇵🇦' },
  { iso2: 'pg', country_name: 'Papua New Guinea', country_code: '675', flag: '🇵🇬' },
  { iso2: 'py', country_name: 'Paraguay', country_code: '595', flag: '🇵🇾' },
  { iso2: 'pe', country_name: 'Peru', country_code: '51', flag: '🇵🇪' },
  { iso2: 'ph', country_name: 'Philippines', country_code: '63', flag: '🇵🇭' },
  { iso2: 'pl', country_name: 'Poland', country_code: '48', flag: '🇵🇱' },
  { iso2: 'pt', country_name: 'Portugal', country_code: '351', flag: '🇵🇹' },
  { iso2: 'pr', country_name: 'Puerto Rico', country_code: '1', flag: '🇵🇷' },
  { iso2: 'qa', country_name: 'Qatar', country_code: '974', flag: '🇶🇦' },
  { iso2: 're', country_name: 'Réunion', country_code: '262', flag: '🇷🇪' },
  { iso2: 'ro', country_name: 'Romania', country_code: '40', flag: '🇷🇴' },
  { iso2: 'ru', country_name: 'Russia', country_code: '7', flag: '🇷🇺' },
  { iso2: 'rw', country_name: 'Rwanda', country_code: '250', flag: '🇷🇼' },
  { iso2: 'ws', country_name: 'Samoa', country_code: '685', flag: '🇼🇸' },
  { iso2: 'sm', country_name: 'San Marino', country_code: '378', flag: '🇸🇲' },
  { iso2: 'st', country_name: 'São Tomé & Príncipe', country_code: '239', flag: '🇸🇹' },
  { iso2: 'sa', country_name: 'Saudi Arabia', country_code: '966', flag: '🇸🇦' },
  { iso2: 'sn', country_name: 'Senegal', country_code: '221', flag: '🇸🇳' },
  { iso2: 'rs', country_name: 'Serbia', country_code: '381', flag: '🇷🇸' },
  { iso2: 'sc', country_name: 'Seychelles', country_code: '248', flag: '🇸🇨' },
  { iso2: 'sl', country_name: 'Sierra Leone', country_code: '232', flag: '🇸🇱' },
  { iso2: 'sg', country_name: 'Singapore', country_code: '65', flag: '🇸🇬' },
  { iso2: 'sx', country_name: 'Sint Maarten', country_code: '1', flag: '🇸🇽' },
  { iso2: 'sk', country_name: 'Slovakia', country_code: '421', flag: '🇸🇰' },
  { iso2: 'si', country_name: 'Slovenia', country_code: '386', flag: '🇸🇮' },
  { iso2: 'sb', country_name: 'Solomon Islands', country_code: '677', flag: '🇸🇧' },
  { iso2: 'so', country_name: 'Somalia', country_code: '252', flag: '🇸🇴' },
  { iso2: 'za', country_name: 'South Africa', country_code: '27', flag: '🇿🇦' },
  { iso2: 'kr', country_name: 'South Korea', country_code: '82', flag: '🇰🇷' },
  { iso2: 'ss', country_name: 'South Sudan', country_code: '211', flag: '🇸🇸' },
  { iso2: 'es', country_name: 'Spain', country_code: '34', flag: '🇪🇸' },
  { iso2: 'lk', country_name: 'Sri Lanka', country_code: '94', flag: '🇱🇰' },
  { iso2: 'bl', country_name: 'St. Barthélemy', country_code: '590', flag: '🇧🇱' },
  { iso2: 'sh', country_name: 'St. Helena', country_code: '290', flag: '🇸🇭' },
  { iso2: 'kn', country_name: 'St. Kitts & Nevis', country_code: '1', flag: '🇰🇳' },
  { iso2: 'lc', country_name: 'St. Lucia', country_code: '1', flag: '🇱🇨' },
  { iso2: 'mf', country_name: 'St. Martin', country_code: '590', flag: '🇲🇫' },
  { iso2: 'pm', country_name: 'St. Pierre & Miquelon', country_code: '508', flag: '🇵🇲' },
  { iso2: 'vc', country_name: 'St. Vincent & Grenadines', country_code: '1', flag: '🇻🇨' },
  { iso2: 'sd', country_name: 'Sudan', country_code: '249', flag: '🇸🇩' },
  { iso2: 'sr', country_name: 'Suriname', country_code: '597', flag: '🇸🇷' },
  { iso2: 'sj', country_name: 'Svalbard & Jan Mayen', country_code: '47', flag: '🇸🇯' },
  { iso2: 'se', country_name: 'Sweden', country_code: '46', flag: '🇸🇪' },
  { iso2: 'ch', country_name: 'Switzerland', country_code: '41', flag: '🇨🇭' },
  { iso2: 'sy', country_name: 'Syria', country_code: '963', flag: '🇸🇾' },
  { iso2: 'tw', country_name: 'Taiwan', country_code: '886', flag: '🇹🇼' },
  { iso2: 'tj', country_name: 'Tajikistan', country_code: '992', flag: '🇹🇯' },
  { iso2: 'tz', country_name: 'Tanzania', country_code: '255', flag: '🇹🇿' },
  { iso2: 'th', country_name: 'Thailand', country_code: '66', flag: '🇹🇭' },
  { iso2: 'tl', country_name: 'Timor-Leste', country_code: '670', flag: '🇹🇱' },
  { iso2: 'tg', country_name: 'Togo', country_code: '228', flag: '🇹🇬' },
  { iso2: 'tk', country_name: 'Tokelau', country_code: '690', flag: '🇹🇰' },
  { iso2: 'to', country_name: 'Tonga', country_code: '676', flag: '🇹🇴' },
  { iso2: 'tt', country_name: 'Trinidad & Tobago', country_code: '1', flag: '🇹🇹' },
  { iso2: 'tn', country_name: 'Tunisia', country_code: '216', flag: '🇹🇳' },
  { iso2: 'tr', country_name: 'Turkey', country_code: '90', flag: '🇹🇷' },
  { iso2: 'tm', country_name: 'Turkmenistan', country_code: '993', flag: '🇹🇲' },
  { iso2: 'tc', country_name: 'Turks & Caicos Islands', country_code: '1', flag: '🇹🇨' },
  { iso2: 'tv', country_name: 'Tuvalu', country_code: '688', flag: '🇹🇻' },
  { iso2: 'vi', country_name: 'U.S. Virgin Islands', country_code: '1', flag: '🇻🇮' },
  { iso2: 'ug', country_name: 'Uganda', country_code: '256', flag: '🇺🇬' },
  { iso2: 'ua', country_name: 'Ukraine', country_code: '380', flag: '🇺🇦' },
  { iso2: 'ae', country_name: 'United Arab Emirates', country_code: '971', flag: '🇦🇪' },
  { iso2: 'gb', country_name: 'United Kingdom', country_code: '44', flag: '🇬🇧' },
  { iso2: 'us', country_name: 'United States', country_code: '1', flag: '🇺🇸' },
  { iso2: 'uy', country_name: 'Uruguay', country_code: '598', flag: '🇺🇾' },
  { iso2: 'uz', country_name: 'Uzbekistan', country_code: '998', flag: '🇺🇿' },
  { iso2: 'vu', country_name: 'Vanuatu', country_code: '678', flag: '🇻🇺' },
  { iso2: 'va', country_name: 'Vatican City', country_code: '39', flag: '🇻🇦' },
  { iso2: 've', country_name: 'Venezuela', country_code: '58', flag: '🇻🇪' },
  { iso2: 'vn', country_name: 'Vietnam', country_code: '84', flag: '🇻🇳' },
  { iso2: 'wf', country_name: 'Wallis & Futuna', country_code: '681', flag: '🇼🇫' },
  { iso2: 'eh', country_name: 'Western Sahara', country_code: '212', flag: '🇪🇭' },
  { iso2: 'ye', country_name: 'Yemen', country_code: '967', flag: '🇾🇪' },
  { iso2: 'zm', country_name: 'Zambia', country_code: '260', flag: '🇿🇲' },
  { iso2: 'zw', country_name: 'Zimbabwe', country_code: '263', flag: '🇿🇼' },
]

// Pre-formatted options for NeCombobox component showing country name, code, and flag
export const countryCodeComboOptions: NeComboboxOption[] = countries.map((c) => ({
  id: `${c.iso2}`,
  label: `${c.country_name} (+${c.country_code})`,
  description: c.flag,
}))

// Parse a raw phone number string and return parts suitable for form fields.
// Used by all phone input components to split phone into country code + local part.
// Returns { countryCode, phone } where:
// - countryCode is ISO2 code (e.g., "it")
// - phone is formatted local part (e.g., "333 000 1113")
// If parsing fails or input is empty, defaults to "it" + original value (or empty).
export function parsePhoneForForm(raw: string | null | undefined): {
  countryCode: string
  phone: string
} {
  if (!raw) {
    return { countryCode: 'it', phone: '' }
  }

  const parsed = parsePhoneNumber(raw)
  if (parsed) {
    return { countryCode: parsed.countryIso2, phone: parsed.localPart }
  } else {
    // Fallback if parsing fails
    return { countryCode: 'it', phone: raw }
  }
}
