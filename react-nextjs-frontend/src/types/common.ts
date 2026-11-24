export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface HealthStatus {
  alive: boolean;
  ready: boolean;
}
