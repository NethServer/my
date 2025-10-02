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
  sortBy: string,
  sortDescending: boolean,
) => {
  return new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    search: textFilter || '',
    sort_by: sortBy,
    sort_direction: sortDescending ? 'desc' : 'asc',
  }).toString()
}

// normalize a string: lowercase and remove spaces
export const normalize = (str: string) => {
  return str.toLowerCase().replace(/\s+/g, '_')
}
