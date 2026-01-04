-- +migrate Up
-- Add new status values to ai_jobs table for summary generation workflow
ALTER TABLE ai_jobs 
DROP CONSTRAINT IF EXISTS ai_jobs_status_check;

ALTER TABLE ai_jobs 
ADD CONSTRAINT ai_jobs_status_check 
CHECK (status IN (
    'pending', 
    'submitted', 
    'processing', 
    'transcript_ready',  
    'summarizing',      
    'completed', 
    'failed', 
    'cancelled'
));

-- +migrate Down
-- Revert to original status constraint
ALTER TABLE ai_jobs 
DROP CONSTRAINT IF EXISTS ai_jobs_status_check;

ALTER TABLE ai_jobs 
ADD CONSTRAINT ai_jobs_status_check 
CHECK (status IN (
    'pending', 
    'submitted', 
    'processing', 
    'completed', 
    'failed', 
    'retrying', 
    'cancelled'
));
