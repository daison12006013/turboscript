-- Enable UUID extension for uuid_generate_v4() function
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Password Security Notice:
-- This database initialization uses Argon2id password hashes for enhanced security.
-- Argon2id is a memory-hard hashing algorithm that provides superior protection
-- against GPU/ASIC attacks compared to traditional algorithms like bcrypt.
-- All sample users have the password "password" for development/testing purposes.

-- Create function for automatic updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    uid UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified_at TIMESTAMP WITH TIME ZONE NULL,
    password VARCHAR(255) NOT NULL,
    remember_token VARCHAR(100) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add updated_at triggers to relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE
    ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample users data with Argon2id password hashes
-- All sample users have the password "password" hashed with Argon2id algorithm
-- Argon2id provides superior security with memory-hard hashing resistant to GPU/ASIC attacks
--
-- MIGRATION NOTE: If upgrading from a previous version with bcrypt/insecure hashes:
-- 1. The application supports gradual migration during user login
-- 2. Old passwords will be automatically upgraded to Argon2id when users log in
-- 3. See app/docs/guides/argon2-migration.md for detailed migration instructions
--
-- Hash Parameters Used:
-- - Algorithm: Argon2id (v=19)
-- - Memory Cost: 65536 KB (64 MB)
-- - Time Cost: 3 iterations
-- - Parallelism: 4 threads
-- - Salt Length: 16 bytes
-- - Hash Length: 32 bytes
INSERT INTO users (uid, name, email, password) VALUES
    ('50d7f275-ecdf-4413-a323-11df86de5fd5', 'John Doe', 'john.doe@example.com', '$argon2id$v=19$m=65536,t=3,p=4$K9lBrQ4mmxB0b/bpwTYFxg$uaUSUEwVjM3Chy0N2m4bdIxDnE718Sls8OR1YoMG/50'), -- password (Argon2id)
    ('3073af78-c857-48e8-a067-97289c467361', 'Jane Smith', 'jane.smith@example.com', '$argon2id$v=19$m=65536,t=3,p=4$gW9gJto+bPIH+w0BnT0TBg$rC+v6AXXlYlY+85zF7/8KnEk5dUeL0N0psAB9xXo1/I'), -- password (Argon2id)
    ('60e3ed9c-a45a-4262-b9c3-f60022307b03', 'Bob Johnson', 'bob.johnson@example.com', '$argon2id$v=19$m=65536,t=3,p=4$7RscOrHHngq6+JPFOcZmdw$6VxBWU+n/3N6R4zydZdd8l/TJ39BhvvIZPSSrhM3pZc') -- password (Argon2id)
ON CONFLICT (email) DO NOTHING;

-- Jobs table for background job processing
CREATE TABLE IF NOT EXISTS jobs (
    uid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(255) NOT NULL,                        -- Job type/handler name (e.g., 'send-confirmation-email')
    path VARCHAR(500) NOT NULL,                        -- TypeScript file path (e.g., 'queue/send-confirmation-email')
    payload JSONB NOT NULL DEFAULT '{}',               -- Job data as JSON
    status VARCHAR(50) NOT NULL DEFAULT 'pending',     -- pending, processing, completed, failed, cancelled
    priority INTEGER NOT NULL DEFAULT 0,               -- Job priority (higher = more priority)
    retry_count INTEGER NOT NULL DEFAULT 0,            -- Current retry attempt count
    max_retries INTEGER NOT NULL DEFAULT 3,            -- Maximum retry attempts allowed
    next_retry_at TIMESTAMP WITH TIME ZONE NULL,       -- When to retry next (NULL if no retry needed)
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP, -- When job was scheduled
    started_at TIMESTAMP WITH TIME ZONE NULL,          -- When job processing started
    completed_at TIMESTAMP WITH TIME ZONE NULL,        -- When job was completed/failed
    error_message TEXT NULL,                           -- Error message if job failed
    error_details JSONB NULL,                          -- Detailed error information as JSON
    timeout_seconds INTEGER NOT NULL DEFAULT 300,      -- Job timeout in seconds
    worker_id VARCHAR(255) NULL,                       -- ID of worker processing the job
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Job history table for tracking all job execution attempts
CREATE TABLE IF NOT EXISTS job_history (
    uid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_uid UUID NOT NULL,                             -- Reference to jobs table
    attempt_number INTEGER NOT NULL,                   -- Which retry attempt this was (1, 2, 3, etc.)
    status VARCHAR(50) NOT NULL,                       -- Status of this attempt: started, completed, failed, timeout
    worker_id VARCHAR(255) NULL,                       -- ID of worker that processed this attempt
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,      -- When this attempt started
    completed_at TIMESTAMP WITH TIME ZONE NULL,        -- When this attempt finished
    duration_ms INTEGER NULL,                          -- How long the attempt took in milliseconds
    error_message TEXT NULL,                           -- Error message if attempt failed
    error_details JSONB NULL,                          -- Detailed error information as JSON
    output JSONB NULL,                                 -- Job output/result data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_uid) REFERENCES jobs(uid) ON DELETE CASCADE
);

-- Add updated_at triggers for jobs table
CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE
    ON jobs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_at ON jobs(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_jobs_next_retry_at ON jobs(next_retry_at) WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_priority_scheduled ON jobs(priority DESC, scheduled_at ASC);
CREATE INDEX IF NOT EXISTS idx_job_history_job_uid ON job_history(job_uid);
CREATE INDEX IF NOT EXISTS idx_job_history_status ON job_history(status);

-- Create configuration table for system settings including retention periods
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add updated_at trigger for system_settings
CREATE TRIGGER update_system_settings_updated_at BEFORE UPDATE
    ON system_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default retention periods
INSERT INTO system_settings (key, value, description) VALUES
    ('job_retention_days', '15', 'Number of days to keep completed, failed, or cancelled jobs'),
    ('job_history_retention_days', '15', 'Number of days to keep job history entries'),
    ('job_auto_cleanup', 'true', 'Whether to automatically clean up old job data')
ON CONFLICT (key) DO NOTHING;

-- Create job retention function to delete old jobs
CREATE OR REPLACE FUNCTION cleanup_old_jobs()
RETURNS integer AS $$
DECLARE
    retention_days INTEGER;
    deleted_count INTEGER;
BEGIN
    -- Get retention period from system_settings
    SELECT COALESCE(NULLIF(value, '')::INTEGER, 15) INTO retention_days
    FROM system_settings
    WHERE key = 'job_retention_days';

    -- Delete old completed, failed, or cancelled jobs
    WITH deleted AS (
        DELETE FROM jobs
        WHERE
            status IN ('completed', 'failed', 'cancelled')
            AND updated_at < (CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL)
        RETURNING uid
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create job history retention function to delete old history entries
CREATE OR REPLACE FUNCTION cleanup_old_job_history()
RETURNS integer AS $$
DECLARE
    retention_days INTEGER;
    deleted_count INTEGER;
BEGIN
    -- Get retention period from system_settings
    SELECT COALESCE(NULLIF(value, '')::INTEGER, 15) INTO retention_days
    FROM system_settings
    WHERE key = 'job_history_retention_days';

    -- Delete old job history entries (while preserving the job record itself)
    WITH deleted AS (
        DELETE FROM job_history
        WHERE
            created_at < (CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL)
            -- Don't delete entries if their job is still in the jobs table
            AND job_uid IN (
                SELECT j.uid FROM jobs j
                WHERE j.status IN ('completed', 'failed', 'cancelled')
            )
        RETURNING uid
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create scheduler executions table for tracking scheduled task execution history
CREATE TABLE IF NOT EXISTS scheduler_executions (
    id SERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL,
    execution_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    jobs_deleted INTEGER DEFAULT 0,
    history_deleted INTEGER DEFAULT 0,
    retention_days_jobs INTEGER DEFAULT 15,
    retention_days_history INTEGER DEFAULT 15,
    execution_duration_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster lookups by task_name and execution_time
CREATE INDEX IF NOT EXISTS idx_scheduler_executions_task_time ON scheduler_executions(task_name, execution_time);
CREATE INDEX IF NOT EXISTS idx_scheduler_executions_created_at ON scheduler_executions(created_at);

-- Add comment for documentation
COMMENT ON TABLE scheduler_executions IS 'Tracks execution history of scheduled tasks, particularly useful for monitoring data retention cleanup operations';
COMMENT ON COLUMN scheduler_executions.task_name IS 'Name of the scheduled task that was executed';
COMMENT ON COLUMN scheduler_executions.execution_time IS 'When the task execution started';
COMMENT ON COLUMN scheduler_executions.jobs_deleted IS 'Number of job records deleted (for cleanup tasks)';
COMMENT ON COLUMN scheduler_executions.history_deleted IS 'Number of job history records deleted (for cleanup tasks)';
COMMENT ON COLUMN scheduler_executions.retention_days_jobs IS 'Retention period used for jobs cleanup';
COMMENT ON COLUMN scheduler_executions.retention_days_history IS 'Retention period used for job history cleanup';
COMMENT ON COLUMN scheduler_executions.execution_duration_ms IS 'Task execution duration in milliseconds';
