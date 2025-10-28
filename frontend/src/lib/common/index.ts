export const MIN_SEARCH_LENGTH = 2

export interface Pagination {
  has_next: boolean
  has_prev: boolean
  page: number
  page_size: number
  total_count: number
  total_pages: number
}

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
