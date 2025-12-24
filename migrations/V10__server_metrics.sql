CREATE TABLE IF NOT EXISTS server_metric_samples (
  id UUID PRIMARY KEY,
  captured_at TIMESTAMPTZ NOT NULL,
  heap_used_bytes BIGINT NOT NULL,
  heap_max_bytes BIGINT NOT NULL,
  system_memory_total_bytes BIGINT NOT NULL,
  system_memory_used_bytes BIGINT NOT NULL,
  process_cpu_load DOUBLE PRECISION,
  system_cpu_load DOUBLE PRECISION
);

CREATE INDEX IF NOT EXISTS idx_server_metric_samples_captured_at ON server_metric_samples (captured_at DESC);
