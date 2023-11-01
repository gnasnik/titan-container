ALTER TABLE services ADD UNIQUE INDEX IF NOT EXISTS uniq_deployment_id_image (`deployment_id`, `image`);
ALTER TABLE services DROP COLUMN IF EXISTS expose_port;