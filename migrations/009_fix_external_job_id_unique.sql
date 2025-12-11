-- +migrate Up
-- Fix external_job_id unique constraint to allow NULLs
-- Problem: UNIQUE constraint in PostgreSQL considers empty strings as duplicate
-- Solution: Drop old unique constraint and create nullable unique constraint

-- Drop old unique constraint if it exists
ALTER TABLE ai_jobs DROP CONSTRAINT IF EXISTS ai_jobs_external_job_id_key;

-- Remove old index
DROP INDEX IF EXISTS idx_ai_jobs_external_id;

-- Set external_job_id to NULL where it's empty string
UPDATE ai_jobs SET external_job_id = NULL WHERE external_job_id = '';

-- Alter column to be nullable
ALTER TABLE ai_jobs ALTER COLUMN external_job_id DROP NOT NULL;

-- Create unique constraint that allows NULLs
-- In PostgreSQL, UNIQUE constraints naturally allow multiple NULLs
ALTER TABLE ai_jobs ADD CONSTRAINT ai_jobs_external_job_id_unique UNIQUE NULLS NOT DISTINCT (external_job_id);

-- Create filtered index for non-null values only
CREATE UNIQUE INDEX idx_ai_jobs_external_id_not_null ON ai_jobs(external_job_id) WHERE external_job_id IS NOT NULL;

-- +migrate Down
-- Rollback migration
ALTER TABLE ai_jobs DROP CONSTRAINT IF EXISTS ai_jobs_external_job_id_unique;
DROP INDEX IF EXISTS idx_ai_jobs_external_id_not_null;
ALTER TABLE ai_jobs ADD CONSTRAINT ai_jobs_external_job_id_key UNIQUE (external_job_id);
ALTER TABLE ai_jobs ALTER COLUMN external_job_id SET NOT NULL;
UPDATE ai_jobs SET external_job_id = '' WHERE external_job_id IS NULL;
CREATE UNIQUE INDEX idx_ai_jobs_external_id ON ai_jobs(external_job_id);
