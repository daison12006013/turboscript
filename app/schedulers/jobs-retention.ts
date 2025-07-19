/**
 * Jobs Retention Cleanup Scheduler
 *
 * This scheduled task handles cleanup of old job records and job history entries
 * based on the configured retention periods. This replaces the built-in Go cleanup
 * logic with a more flexible TypeScript-based approach that developers can customize.
 *
 * Schedule: Runs daily at 2:00 AM UTC (configurable in turboscript.yml)
 *
 * Payload options:
 * - cleanup_type: "jobs" | "history" | "all" (default: "all")
 * - dry_run: boolean (default: false) - if true, shows what would be deleted without deleting
 * - force_cleanup: boolean (default: false) - if true, ignores retention settings and cleans based on payload
 * - jobs_days: number (optional) - override retention period for jobs
 * - history_days: number (optional) - override retention period for history
 */

interface SchedulerPayload {
    cleanup_type?: 'jobs' | 'history' | 'all';
    dry_run?: boolean;
    force_cleanup?: boolean;
    jobs_days?: number;
    history_days?: number;
}

interface SchedulerEvent extends Event {
    task_name: string;
    cron: string;
    timezone: string;
    run_count: number;
    scheduled_time: string;
    payload: SchedulerPayload;
}

interface CleanupResult {
    jobs_deleted: number;
    history_deleted: number;
    jobs_retention_days: number;
    history_retention_days: number;
    dry_run: boolean;
    execution_time_ms: number;
}

export const handle = async (event: SchedulerEvent): Promise<void> => {
    const startTime = Date.now();
    const payload = event.payload;
    const cleanupType = payload.cleanup_type ?? 'all';
    const isDryRun = payload.dry_run ?? false;

    console.log(`[${event.task_name}] Starting jobs retention cleanup`, {
        cleanup_type: cleanupType,
        dry_run: isDryRun,
        run_count: event.run_count,
        scheduled_time: event.scheduled_time
    });

    try {
        // Get retention settings from database system_settings or use payload overrides
        const retentionSettings = await getRetentionSettings(payload);

        const result: CleanupResult = {
            jobs_deleted: 0,
            history_deleted: 0,
            jobs_retention_days: retentionSettings.jobs_days,
            history_retention_days: retentionSettings.history_days,
            dry_run: isDryRun,
            execution_time_ms: 0
        };

        // Clean up jobs if requested
        if (cleanupType === 'jobs' || cleanupType === 'all') {
            result.jobs_deleted = await cleanupOldJobs(retentionSettings.jobs_days, isDryRun);
        }

        // Clean up job history if requested
        if (cleanupType === 'history' || cleanupType === 'all') {
            result.history_deleted = await cleanupOldJobHistory(retentionSettings.history_days, isDryRun);
        }

        result.execution_time_ms = Date.now() - startTime;

        // Log results
        const action = isDryRun ? 'Would clean up' : 'Cleaned up';
        console.log(`[${event.task_name}] ${action}:`, {
            jobs_deleted: result.jobs_deleted,
            history_deleted: result.history_deleted,
            jobs_retention_days: result.jobs_retention_days,
            history_retention_days: result.history_retention_days,
            execution_time_ms: result.execution_time_ms
        });

        // Store cleanup statistics for monitoring
        if (!isDryRun && (result.jobs_deleted > 0 || result.history_deleted > 0)) {
            await logCleanupExecution(event.task_name, result);
        }

    } catch (error) {
        console.error(`[${event.task_name}] Jobs retention cleanup failed:`, error);
        throw error;
    }
};

/**
 * Get retention settings from database system_settings table or use payload overrides
 */
async function getRetentionSettings(payload: SchedulerPayload): Promise<{ jobs_days: number; history_days: number }> {
    // Use payload overrides if provided
    if (payload.force_cleanup && (payload.jobs_days ?? payload.history_days)) {
        return {
            jobs_days: payload.jobs_days ?? 15,
            history_days: payload.history_days ?? 15
        };
    }

    try {
        // Get retention settings from database
        const jobsResult = await turboQuery<{ retention_days: number }>(
            "SELECT COALESCE(NULLIF(value, '')::INTEGER, 15) as retention_days FROM system_settings WHERE key = $1",
            ['job_retention_days']
        );

        const historyResult = await turboQuery<{ retention_days: number }>(
            "SELECT COALESCE(NULLIF(value, '')::INTEGER, 15) as retention_days FROM system_settings WHERE key = $1",
            ['job_history_retention_days']
        );

        return {
            jobs_days: jobsResult.length > 0 ? Number(jobsResult[0].retention_days) : 15,
            history_days: historyResult.length > 0 ? Number(historyResult[0].retention_days) : 15
        };
    } catch (error) {
        console.warn('Failed to get retention settings from database, using defaults:', error);
        return {
            jobs_days: 15,
            history_days: 15
        };
    }
}

/**
 * Clean up old job records
 */
async function cleanupOldJobs(retentionDays: number, isDryRun: boolean): Promise<number> {
    try {
        if (isDryRun) {
            // Count what would be deleted
            const countResult = await turboQuery<{ count: number }>(`
                SELECT COUNT(*) as count
                FROM jobs
                WHERE status IN ('completed', 'failed', 'cancelled')
                AND updated_at < (CURRENT_TIMESTAMP - ($1 || ' days')::INTERVAL)
            `, [retentionDays.toString()]);

            return countResult.length > 0 ? Number(countResult[0]?.count) : 0;
        } else {
            // Use the database cleanup function
            const result = await turboQuery<{ deleted_count: number }>('SELECT cleanup_old_jobs() as deleted_count');
            return result.length > 0 ? Number(result[0]?.deleted_count) : 0;
        }
    } catch (error) {
        console.error('Failed to cleanup old jobs:', error);
        throw error;
    }
}

/**
 * Clean up old job history records
 */
async function cleanupOldJobHistory(retentionDays: number, isDryRun: boolean): Promise<number> {
    try {
        if (isDryRun) {
            // Count what would be deleted
            const countResult = await turboQuery<{ count: number }>(`
                SELECT COUNT(*) as count
                FROM job_history
                WHERE created_at < (CURRENT_TIMESTAMP - ($1 || ' days')::INTERVAL)
                AND job_uid IN (
                    SELECT j.uid FROM jobs j
                    WHERE j.status IN ('completed', 'failed', 'cancelled')
                    AND j.updated_at < (CURRENT_TIMESTAMP - ($1 || ' days')::INTERVAL)
                )
            `, [retentionDays.toString()]);

            return countResult.length > 0 ? Number(countResult[0]?.count) : 0;
        } else {
            // Use the database cleanup function
            const result = await turboQuery<{ deleted_count: number }>('SELECT cleanup_old_job_history() as deleted_count');
            return result.length > 0 ? Number(result[0].deleted_count) : 0;
        }
    } catch (error) {
        console.error('Failed to cleanup old job history:', error);
        throw error;
    }
}

/**
 * Log cleanup execution for monitoring and statistics
 */
async function logCleanupExecution(taskName: string, result: CleanupResult): Promise<void> {
    try {
        await turboQuery(`
            INSERT INTO scheduler_executions (task_name, execution_time, jobs_deleted, history_deleted, retention_days_jobs, retention_days_history, execution_duration_ms)
            VALUES ($1, CURRENT_TIMESTAMP, $2, $3, $4, $5, $6)
        `, [
            taskName,
            result.jobs_deleted,
            result.history_deleted,
            result.jobs_retention_days,
            result.history_retention_days,
            result.execution_time_ms
        ]);
    } catch (error) {
        // Don't fail the cleanup if logging fails
        console.warn('Failed to log cleanup execution (this is not critical):', error);
    }
}
