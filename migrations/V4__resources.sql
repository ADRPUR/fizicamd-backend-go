CREATE TABLE IF NOT EXISTS resource_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  group_label TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS resource_entries (
  id UUID PRIMARY KEY,
  category_code TEXT NOT NULL REFERENCES resource_categories(code) ON DELETE RESTRICT,
  author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  summary TEXT NOT NULL,
  avatar_media_id UUID NULL REFERENCES media_assets(id) ON DELETE SET NULL,
  content JSONB NOT NULL DEFAULT '[]'::jsonb,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL,
  published_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_entries_category_status ON resource_entries(category_code, status);
CREATE INDEX IF NOT EXISTS idx_resource_entries_status_published ON resource_entries(status, published_at DESC);

INSERT INTO resource_categories (code, label, group_label, sort_order) VALUES
  ('bac-fizica', 'BAC_Fizica', 'Olimpiade', 1),
  ('baraj-fizica', 'Baraj Fizica', 'Olimpiade', 2),
  ('baraj-stiinte-juniori', 'Baraj Științe Juniori', 'Olimpiade', 3),
  ('cfte-galileo', 'C.F.T.E. "Galileo"', 'Olimpiade', 4),
  ('concurs-municipal-fizica-juniori', 'Concurs Municipal Fizica Juniori (cl 6-8)', 'Olimpiade', 5),
  ('etapa-sector-chisinau', 'Etapa de Sector Chișinău (cl 9-12)', 'Olimpiade', 6),
  ('etapa-municipala-chisinau', 'Etapa Municipală Chișinău (cl 9-12)', 'Olimpiade', 7),
  ('impuls', 'Impuls (cl.VI-IX)', 'Olimpiade', 8),
  ('in-memoriam-mihai-marinicuc', 'În memoriam Mihai Marinicuc', 'Olimpiade', 9),
  ('olimpiada-internationala-fizica', 'Olimpiada Internațională Fizica', 'Olimpiade', 10),
  ('olimpiada-republicana', 'Olimpiada Republicană', 'Olimpiade', 11),
  ('orarul-olimpiadelor-2019-2020', 'Orarul Olimpiadelor 2019-2020', 'Olimpiade', 12),
  ('stiinte-municipiu', 'ȘTIINȚE(municipiu)', 'Olimpiade', 13),
  ('cercul-de-fizica-quant', 'Cercul de fizică "Quant"', 'Categorii', 1),
  ('evaluarea', 'Evaluarea', 'Categorii', 2),
  ('gaudeamus', 'Gaudeamus', 'Categorii', 3),
  ('metodica', 'Metodica', 'Categorii', 4),
  ('noutati-in-fizica', 'Noutăți în fizică', 'Categorii', 5),
  ('olimpiade', 'Olimpiade', 'Categorii', 6),
  ('orizont-ciocana', 'Orizont, Ciocana', 'Categorii', 7),
  ('tabara-olimpica', 'Tabăra olimpică', 'Categorii', 8),
  ('uncategorized', 'Uncategorized', 'Categorii', 9),
  ('scoala-de-vara-lt-gaudeamus', 'Școala de vară la L.T. Gaudeamus', 'Categorii', 10),
  ('carti-in-l-rusa', 'Cărți în l.rusă', 'Reviste', 1),
  ('zhurnal-kvant', 'Журнал Квант', 'Reviste', 2),
  ('revista-ftm', 'Revista FTM', 'Reviste', 3)
ON CONFLICT (code) DO NOTHING;
