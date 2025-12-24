CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  status TEXT NOT NULL,
  is_email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ NULL,
  deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_lower ON users ((lower(email)));

CREATE TABLE IF NOT EXISTS roles (
  id UUID PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  description TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS media_assets (
  id UUID PRIMARY KEY,
  owner_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  bucket TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  filename TEXT NULL,
  description TEXT NULL,
  type TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256 TEXT NULL,
  access_policy TEXT NOT NULL DEFAULT 'PRIVATE',
  status TEXT NOT NULL DEFAULT 'READY',
  metadata JSONB NULL DEFAULT '{}'::jsonb,
  expires_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_media_assets_owner ON media_assets(owner_user_id);

CREATE TABLE IF NOT EXISTS user_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  first_name TEXT NULL,
  last_name TEXT NULL,
  birth_date DATE NULL,
  gender TEXT NULL,
  phone TEXT NULL,
  school TEXT NULL,
  grade_level TEXT NULL,
  locale TEXT NULL,
  timezone TEXT NULL,
  bio TEXT NULL,
  avatar_media_id UUID NULL REFERENCES media_assets(id) ON DELETE SET NULL,
  contact_json JSONB NULL DEFAULT '{}'::jsonb,
  metadata JSONB NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_roles (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_user_role UNIQUE (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS groups (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NULL,
  grade INT NULL,
  year INT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  teacher_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS group_members (
  id UUID PRIMARY KEY,
  group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  member_role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uk_group_members_group_user UNIQUE (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_group ON group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members(user_id);

CREATE TABLE IF NOT EXISTS navigation_items (
  id UUID PRIMARY KEY,
  parent_id UUID NULL REFERENCES navigation_items(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  type TEXT NOT NULL,
  href TEXT NULL,
  visibility TEXT NOT NULL DEFAULT 'PUBLIC',
  status TEXT NOT NULL DEFAULT 'DRAFT',
  sort_order INT NOT NULL DEFAULT 0,
  icon TEXT NULL,
  payload JSONB NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_navigation_parent ON navigation_items(parent_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_navigation_slug_visibility ON navigation_items(slug, visibility);
