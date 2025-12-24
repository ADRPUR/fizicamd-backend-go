CREATE TABLE IF NOT EXISTS site_visits (
  id UUID PRIMARY KEY,
  ip_address TEXT NULL,
  user_agent TEXT NULL,
  path TEXT NULL,
  referrer TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_site_visits_created_at ON site_visits(created_at);
