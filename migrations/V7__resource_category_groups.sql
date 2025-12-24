ALTER TABLE resource_categories
  ADD COLUMN IF NOT EXISTS group_order INT NOT NULL DEFAULT 0;

UPDATE resource_categories
SET group_order = CASE group_label
  WHEN 'Olimpiade' THEN 1
  WHEN 'Categorii' THEN 2
  WHEN 'Reviste' THEN 3
  ELSE 4
END
WHERE group_order = 0;
