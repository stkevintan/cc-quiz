export interface HealthResponse {
  status: 'ok' | 'degraded' | string;
  service: string;
  environment: string;
  time: string;
  database: {
    status: 'ok' | 'error' | string;
    appliedMigrations: number;
    error?: string;
  };
}
