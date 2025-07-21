/**
 * Daily Analytics Scheduler Example
 *
 * This is an example scheduled task that demonstrates how to create
 * additional schedulers for various maintenance and processing tasks.
 *
 * Schedule: Daily at 3:00 AM UTC (configurable in turboscript.yml)
 *
 * To enable this scheduler:
 * 1. Uncomment the daily-analytics task in turboscript.yml
 * 2. Modify the payload and schedule as needed
 * 3. Implement your specific analytics logic below
 *
 * Payload options:
 * - date: "yesterday" | "today" | specific date (YYYY-MM-DD)
 * - metrics: string[] - which metrics to process
 * - email_report: boolean - whether to send email report
 */

interface AnalyticsPayload {
    date?: string;
    metrics?: string[];
    email_report?: boolean;
}

interface SchedulerEvent extends Event {
    task_name: string;
    cron: string;
    timezone: string;
    run_count: number;
    scheduled_time: string;
    payload: AnalyticsPayload;
}

interface AnalyticsResult {
    date: string;
    total_users: number;
    new_users: number;
    total_jobs: number;
    successful_jobs: number;
    failed_jobs: number;
    avg_response_time_ms: number;
    processed_at: string;
}

export const handle = async (event: SchedulerEvent): Promise<void> => {
    const startTime = Date.now();
    const { payload } = event;

    console.log(`[${event.task_name}] Starting daily analytics processing`, {
        date: payload.date,
        metrics: payload.metrics,
        email_report: payload.email_report,
        run_count: event.run_count,
        scheduled_time: event.scheduled_time
    });

    try {
        // Determine the date to process
        const targetDate = getTargetDate(payload.date);

        // Process analytics for the target date
        const result = await processAnalytics(targetDate, payload.metrics);

        // Store results in database
        await storeAnalyticsResult(result);

        // Send email report if requested
        if (payload.email_report) {
            await sendAnalyticsReport(result);
        }

        const executionTime = Date.now() - startTime;
        console.log(`[${event.task_name}] Analytics processing completed`, {
            date: result.date,
            total_users: result.total_users,
            new_users: result.new_users,
            total_jobs: result.total_jobs,
            execution_time_ms: executionTime
        });

    } catch (error) {
        console.error(`[${event.task_name}] Analytics processing failed:`, error);
        throw error;
    }
};

/**
 * Determine the target date for analytics processing
 */
function getTargetDate(dateInput?: string): string {
    if (!dateInput || dateInput === 'yesterday') {
        const yesterday = new Date();
        yesterday.setDate(yesterday.getDate() - 1);
        return yesterday.toISOString().split('T')[0];
    }

    if (dateInput === 'today') {
        return new Date().toISOString().split('T')[0];
    }

    // Validate date format (YYYY-MM-DD)
    if (!/^\d{4}-\d{2}-\d{2}$/.test(dateInput)) {
        throw new Error(`Invalid date format: ${dateInput}. Expected YYYY-MM-DD`);
    }

    return dateInput;
}

/**
 * Process analytics for the specified date
 */
async function processAnalytics(date: string, metrics?: string[]): Promise<AnalyticsResult> {
    const allMetrics = metrics ?? ['users', 'jobs', 'performance'];

    const result: AnalyticsResult = {
        date,
        total_users: 0,
        new_users: 0,
        total_jobs: 0,
        successful_jobs: 0,
        failed_jobs: 0,
        avg_response_time_ms: 0,
        processed_at: new Date().toISOString()
    };

    // Process user metrics
    if (allMetrics.includes('users')) {
        const userMetrics = await processUserMetrics(date);
        result.total_users = userMetrics.total;
        result.new_users = userMetrics.new;
    }

    // Process job metrics
    if (allMetrics.includes('jobs')) {
        const jobMetrics = await processJobMetrics(date);
        result.total_jobs = jobMetrics.total;
        result.successful_jobs = jobMetrics.successful;
        result.failed_jobs = jobMetrics.failed;
    }

    return result;
}

/**
 * Process user-related metrics
 */
async function processUserMetrics(date: string): Promise<{ total: number; new: number }> {
    try {
        // Get total users as of the date
        const totalResult = await turboQuery<{ count: number }>(`
            SELECT COUNT(*) as count
            FROM users
            WHERE created_at <= $1::date + INTERVAL '1 day'
        `, [date]);

        // Get new users on the date
        const newResult = await turboQuery<{ count: number }>(`
            SELECT COUNT(*) as count
            FROM users
            WHERE DATE(created_at) = $1
        `, [date]);

        return {
            total: totalResult[0]?.count ?? 0,
            new: newResult[0]?.count ?? 0
        };
    } catch (error) {
        console.error('Failed to process user metrics:', error);
        return { total: 0, new: 0 };
    }
}

/**
 * Process job-related metrics
 */
async function processJobMetrics(date: string): Promise<{ total: number; successful: number; failed: number }> {
    try {
        const jobsResult = await turboQuery<{ total: number; successful: number; failed: number }>(`
            SELECT
                COUNT(*) as total,
                COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful,
                COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
            FROM jobs
            WHERE DATE(created_at) = $1
        `, [date]);

        return {
            total: jobsResult[0]?.total ?? 0,
            successful: jobsResult[0]?.successful ?? 0,
            failed: jobsResult[0]?.failed ?? 0,
        };
    } catch (error) {
        console.error('Failed to process job metrics:', error);
        return { total: 0, successful: 0, failed: 0 };
    }
}

/**
 * Store analytics results in database
 */
async function storeAnalyticsResult(result: AnalyticsResult): Promise<void> {
    try {
        // Create daily_analytics table if it doesn't exist
        await turboQuery(`
            CREATE TABLE IF NOT EXISTS daily_analytics (
                id SERIAL PRIMARY KEY,
                date DATE NOT NULL UNIQUE,
                total_users INTEGER DEFAULT 0,
                new_users INTEGER DEFAULT 0,
                total_jobs INTEGER DEFAULT 0,
                successful_jobs INTEGER DEFAULT 0,
                failed_jobs INTEGER DEFAULT 0,
                avg_response_time_ms DECIMAL(10,2) DEFAULT 0,
                processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
            )
        `);

        // Insert or update the analytics result
        await turboQuery(`
            INSERT INTO daily_analytics (
                date, total_users, new_users, total_jobs, successful_jobs,
                failed_jobs, avg_response_time_ms, processed_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            ON CONFLICT (date) DO UPDATE SET
                total_users = EXCLUDED.total_users,
                new_users = EXCLUDED.new_users,
                total_jobs = EXCLUDED.total_jobs,
                successful_jobs = EXCLUDED.successful_jobs,
                failed_jobs = EXCLUDED.failed_jobs,
                avg_response_time_ms = EXCLUDED.avg_response_time_ms,
                processed_at = EXCLUDED.processed_at
        `, [
            result.date,
            result.total_users,
            result.new_users,
            result.total_jobs,
            result.successful_jobs,
            result.failed_jobs,
            result.avg_response_time_ms,
            result.processed_at
        ]);

        console.log(`Stored analytics for ${result.date}`);
    } catch (error) {
        console.error('Failed to store analytics result:', error);
        throw error;
    }
}

/**
 * Send analytics report via email
 */
async function sendAnalyticsReport(result: AnalyticsResult): Promise<void> {
    try {
        const htmlContent = `
            <h2>Daily Analytics Report - ${result.date}</h2>
            <table style="border-collapse: collapse; width: 100%;">
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>Total Users:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.total_users.toLocaleString()}</td>
                </tr>
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>New Users:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.new_users.toLocaleString()}</td>
                </tr>
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>Total Jobs:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.total_jobs.toLocaleString()}</td>
                </tr>
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>Successful Jobs:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.successful_jobs.toLocaleString()}</td>
                </tr>
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>Failed Jobs:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.failed_jobs.toLocaleString()}</td>
                </tr>
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;"><strong>Avg Response Time:</strong></td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${result.avg_response_time_ms.toFixed(2)}ms</td>
                </tr>
            </table>
            <p><small>Report generated at: ${result.processed_at}</small></p>
        `;

        await turboEmail({
            to: 'admin@example.com', // Configure this in your scheduler task payload
            subject: `Daily Analytics Report - ${result.date}`,
            content: htmlContent,
            html: htmlContent,
            driver: 'smtp'
        });

        console.log(`Analytics report sent for ${result.date}`);
    } catch (error) {
        console.error('Failed to send analytics report:', error);
        // Don't throw error here as email failure shouldn't fail the entire analytics processing
    }
}
