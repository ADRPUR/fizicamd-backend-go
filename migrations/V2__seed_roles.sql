INSERT INTO roles (id, code, description) VALUES
  (uuid_generate_v4(), 'ADMIN', 'Platform administrator'),
  (uuid_generate_v4(), 'TEACHER', 'Educator / content author'),
  (uuid_generate_v4(), 'STUDENT', 'Learner account')
ON CONFLICT (code) DO NOTHING;
