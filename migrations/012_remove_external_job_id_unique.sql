-- +migrate Up
-- Remove UNIQUE constraint from external_job_id to allow multiple NULL values
-- Multiple AI jobs can be created without external_job_id initially
ALTER TABLE ai_jobs DROP CONSTRAINT IF EXISTS ai_jobs_external_job_id_unique;
ALTER TABLE ai_jobs DROP CONSTRAINT IF EXISTS ai_jobs_external_job_id_key;

-- Create a partial unique index that only applies to non-NULL values
-- This ensures external_job_id is unique when it has a value, but allows multiple NULLs
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_jobs_external_job_id_unique 
ON ai_jobs(external_job_id) 
WHERE external_job_id IS NOT NULL;

-- +migrate Down
-- Restore the original unique constraint (if needed)
DROP INDEX IF EXISTS idx_ai_jobs_external_job_id_unique;
ALTER TABLE ai_jobs ADD CONSTRAINT ai_jobs_external_job_id_unique UNIQUE (external_job_id);
