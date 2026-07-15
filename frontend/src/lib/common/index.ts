export const MIN_SEARCH_LENGTH = 2

export const OPTIONS_PAGE_SIZE = 50

export interface Pagination {
  has_next: boolean
  has_prev: boolean
  page: number
  page_size: number
  total_count: number
  total_pages: number
}

export type Focusable = { focus(): void }

// utility to build query string params for pagination, filtering and sorting
export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
  }
  return searchParams.toString()
}

// normalize a string: lowercase and remove spaces
export const normalize = (str: string) => {
  return str.toLowerCase().replace(/\s+/g, '_')
}

export interface TextToken {
  type: 'text' | 'url'
  value: string
}

// Splits a free-form string into ordered text/URL tokens, so callers can
// render the text with the URLs turned into inline links. Only http/https
// URLs (validated via the URL constructor, trailing sentence punctuation
// stripped) become `url` tokens, so their value is safe to bind to an anchor
// href without risking javascript:/data: injection; anything else stays inside
// `text` tokens.
export const tokenizeText = (text: string): TextToken[] => {
  if (!text) return []

  const tokens: TextToken[] = []
  let lastIndex = 0

  for (const match of text.matchAll(/\bhttps?:\/\/[^\s<>"']+/gi)) {
    // Trailing punctuation is usually sentence punctuation, not part of the URL.
    const raw = match[0].replace(/[.,;:!?)\]}]+$/, '')
    const start = match.index

    try {
      const url = new URL(raw)
      if (url.protocol !== 'http:' && url.protocol !== 'https:') continue
    } catch {
      // Not a valid URL — leave it as plain text.
      continue
    }

    if (start > lastIndex) {
      tokens.push({ type: 'text', value: text.slice(lastIndex, start) })
    }
    tokens.push({ type: 'url', value: raw })
    lastIndex = start + raw.length
  }

  if (lastIndex < text.length) {
    tokens.push({ type: 'text', value: text.slice(lastIndex) })
  }

  return tokens
}

// Extracts the http(s) URLs found in a free-form string, in order and
// de-duplicated. Safe to bind to an anchor href (see tokenizeText).
export const extractUrls = (text: string): string[] => {
  const seen = new Set<string>()
  const urls: string[] = []

  for (const token of tokenizeText(text)) {
    if (token.type === 'url' && !seen.has(token.value)) {
      seen.add(token.value)
      urls.push(token.value)
    }
  }

  return urls
}

export const abbreviateNumber = (
  value: number,
  locale = navigator.language,
  minValue = 10_000,
  decimals = 1,
): string => {
  if (Math.abs(value) < minValue) {
    return String(value)
  }
  return new Intl.NumberFormat(locale, {
    notation: 'compact',
    minimumFractionDigits: 0,
    maximumFractionDigits: decimals,
  }).format(value)
}

export const downloadFile = (fileData: string, filename: string, type: 'pdf' | 'csv') => {
  const mimeType = type === 'pdf' ? 'application/pdf' : 'text/csv;charset=utf-8;'

  // Convert the PDF string to a Blob
  const blob = new Blob([fileData], { type: mimeType })

  // Create a download link
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = filename

  // Trigger the download
  link.click()

  // Clean up the URL object
  URL.revokeObjectURL(link.href)
}
