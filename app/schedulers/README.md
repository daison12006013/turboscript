# TurboScript Schedulers

The TurboScript scheduler system allows you to run TypeScript functions on a recurring schedule using cron expressions. This is perfect for maintenance tasks, data processing, report generation, and cleanup operations.

## Overview

Schedulers are TypeScript files that export a `handle` function, similar to background jobs but triggered by time rather than manual dispatch. They run in isolated TypeScript execution environments and have access to all TurboScript runtime functions like `turboQuery`, `turboEmail`, etc.

## Configuration

Schedulers are configured in `turboscript.yml` under the `scheduler` section:

```yaml
scheduler:
  enabled: true
  timezone: "UTC"
  path: "./app/schedulers"
  log_level: "info"

  tasks:
    - name: "jobs-retention-cleanup"
      cron: "0 2 * * *"                    # Daily at 2:00 AM UTC
      handler: "jobs-retention.ts"
      enabled: true
      timezone: "UTC"
      timeout: 300                         # 5 minutes timeout
      description: "Clean up old jobs and job history records"
      payload:
        cleanup_type: "all"
        dry_run: false
      environment:
        CLEANUP_LOG_LEVEL: "info"
```

## Configuration Fields

### Global Scheduler Settings

- `enabled`: Whether the scheduler system is enabled (default: true)
- `timezone`: Default timezone for all tasks (default: "UTC")
- `path`: Directory containing scheduler handlers (default: "./app/schedulers")
- `log_level`: Logging level for scheduler operations (debug, info, warn, error)

### Task Configuration

- `name`: Unique task name (required)
- `cron`: Cron expression defining when to run (required)
- `handler`: TypeScript file relative to scheduler path (required)
- `enabled`: Whether this specific task is enabled (default: true)
- `timezone`: Timezone for this task (overrides global timezone)
- `timeout`: Task timeout in seconds (overrides global timeout)
- `description`: Optional description for documentation
- `payload`: Static data passed to the handler
- `environment`: Environment variables specific to this task

## Cron Expression Format

TurboScript uses 6-field cron expressions with seconds support:

```
# 6-field format (required)
# ┌───────────── second (0 - 59)
# │ ┌───────────── minute (0 - 59)
# │ │ ┌───────────── hour (0 - 23)
# │ │ │ ┌───────────── day of the month (1 - 31)
# │ │ │ │ ┌───────────── month (1 - 12)
# │ │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday)
# │ │ │ │ │ │
# * * * * * *
```

### Common Cron Examples

```yaml
# Every minute
cron: "0 * * * * *"

# Every hour at minute 0
cron: "0 0 * * * *"

# Daily at 2:30 AM
cron: "0 30 2 * * *"

# Weekly on Monday at 1:00 AM
cron: "0 0 1 * * 1"

# Monthly on the 1st at midnight
cron: "0 0 0 1 * *"

# Every 15 minutes
cron: "0 */15 * * * *"

# Business hours (9 AM to 5 PM) on weekdays
cron: "0 0 9-17 * * 1-5"

# Every 30 seconds
cron: "*/30 * * * * *"
```

## Handler Function

Each scheduler handler must export a `handle` function with the following signature:

```typescript
interface SchedulerEvent extends Event {
    task_name: string;
    cron: string;
    timezone: string;
    run_count: number;
    scheduled_time: string;
    payload: any; // Your custom payload type
}

export const handle = async (event: SchedulerEvent): Promise<void> => {
    // Your scheduler logic here
};
```

### Event Properties

- `task_name`: Name of the scheduled task
- `cron`: Cron expression for this task
- `timezone`: Timezone the task is running in
- `run_count`: Number of times this task has been executed
- `scheduled_time`: ISO timestamp when this execution was scheduled
- `payload`: Static payload from configuration
- `env`: Environment variables (global + task-specific)

## Available Runtime Functions

Schedulers have access to all TurboScript runtime functions:

### Database Operations

```typescript
// Query database
const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);

// Use specific database connection
const analytics = await turboQuery({
    query: 'SELECT COUNT(*) FROM page_views WHERE date = $1',
    bindings: ['2024-01-01'],
    connection: 'analytics'
});
```

### Email Operations

```typescript
await turboEmail({
    to: 'admin@example.com',
    subject: 'Daily Report',
    content: '<h1>Report Content</h1>',
    driver: 'smtp'
});
```

### Background Jobs

```typescript
// Dispatch a background job
const jobId = await turboJob('send-notification.ts', {
    userId: 123,
    message: 'Hello World'
});
```

### File Operations

```typescript
// Process HTML templates
const html = await turboHtml('./templates/report.html', { data: analyticsData });

// Process Markdown
const content = await turboMarkdownHtml('./docs/readme.md', { version: '1.0' });
```

## Built-in Schedulers

### Jobs Retention Cleanup (`jobs-retention.ts`)

Automatically cleans up old job records and job history based on retention policies. This replaces the built-in Go cleanup logic with a flexible TypeScript approach.

**Configuration:**

```yaml
- name: "jobs-retention-cleanup"
  cron: "0 2 * * *"                    # Daily at 2:00 AM
  handler: "jobs-retention.ts"
  enabled: true
  timeout: 300                         # 5 minutes
  payload:
    cleanup_type: "all"                # "jobs", "history", "all"
    dry_run: false                     # Set to true to preview deletions
```

**Payload Options:**

- `cleanup_type`: Which records to clean ("jobs", "history", "all")
- `dry_run`: Preview what would be deleted without actually deleting
- `force_cleanup`: Ignore retention settings and use payload overrides
- `jobs_days`: Override retention period for jobs
- `history_days`: Override retention period for history

## Creating Custom Schedulers

### 1. Create Handler File

Create a TypeScript file in `app/schedulers/`:

```typescript
// app/schedulers/weekly-reports.ts

interface ReportPayload {
    recipients: string[];
    report_type: string;
}

interface SchedulerEvent extends Event {
    task_name: string;
    cron: string;
    timezone: string;
    run_count: number;
    scheduled_time: string;
    payload: ReportPayload;
}

export const handle = async (event: SchedulerEvent): Promise<void> => {
    console.log(`Running ${event.task_name} - execution #${event.run_count}`);

    try {
        // Get data for the report
        const data = await turboQuery(`
            SELECT COUNT(*) as total_users,
                   COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as new_users
            FROM users
        `);

        // Generate report
        const reportHtml = await turboHtml('./templates/weekly-report.html', {
            data: data[0],
            week_ending: new Date().toISOString().split('T')[0]
        });

        // Send to recipients
        for (const recipient of event.payload.recipients) {
            await turboEmail({
                to: recipient,
                subject: `Weekly Report - ${new Date().toISOString().split('T')[0]}`,
                content: reportHtml,
                driver: 'smtp'
            });
        }

        console.log(`Weekly report sent to ${event.payload.recipients.length} recipients`);

    } catch (error) {
        console.error('Weekly report failed:', error);
        throw error;
    }
};
```

### 2. Add to Configuration

Add the task to `turboscript.yml`:

```yaml
scheduler:
  tasks:
    - name: "weekly-reports"
      cron: "0 9 * * 1"                    # Monday at 9:00 AM
      handler: "weekly-reports.ts"
      enabled: true
      timezone: "America/New_York"
      timeout: 600                         # 10 minutes
      description: "Generate and send weekly reports"
      payload:
        recipients:
          - "admin@example.com"
          - "manager@example.com"
        report_type: "summary"
      environment:
        REPORT_DEBUG: "false"
```

## Best Practices

### Error Handling

```typescript
export const handle = async (event: SchedulerEvent): Promise<void> => {
    try {
        // Your main logic here
        await processData();

    } catch (error) {
        // Log error with context
        console.error(`[${event.task_name}] Failed:`, {
            error: error.message,
            run_count: event.run_count,
            scheduled_time: event.scheduled_time
        });

        // Re-throw to mark the execution as failed
        throw error;
    }
};
```

### Logging

```typescript
// Use structured logging
console.log(`[${event.task_name}] Starting processing`, {
    run_count: event.run_count,
    payload: event.payload
});

// Log progress for long-running tasks
console.log(`[${event.task_name}] Processed ${count} records`);

// Log completion with metrics
console.log(`[${event.task_name}] Completed`, {
    records_processed: count,
    execution_time_ms: Date.now() - startTime
});
```

### Database Operations

```typescript
// Use transactions for data consistency
await turboQuery('BEGIN');
try {
    await turboQuery('UPDATE table1 SET status = $1', ['processed']);
    await turboQuery('INSERT INTO table2 VALUES ($1, $2)', [data1, data2]);
    await turboQuery('COMMIT');
} catch (error) {
    await turboQuery('ROLLBACK');
    throw error;
}

// Use parameterized queries
const results = await turboQuery(
    'SELECT * FROM events WHERE date >= $1 AND date < $2',
    [startDate, endDate]
);
```

### Performance

```typescript
// Process data in batches for large datasets
const batchSize = 1000;
let offset = 0;

while (true) {
    const batch = await turboQuery(
        'SELECT * FROM large_table LIMIT $1 OFFSET $2',
        [batchSize, offset]
    );

    if (batch.length === 0) break;

    await processBatch(batch);
    offset += batchSize;

    // Log progress
    console.log(`Processed ${offset} records`);
}
```

## Monitoring

### Execution History

The scheduler automatically tracks execution history in the `scheduler_executions` table:

```sql
SELECT
    task_name,
    execution_time,
    execution_duration_ms,
    jobs_deleted,
    history_deleted
FROM scheduler_executions
WHERE task_name = 'jobs-retention-cleanup'
ORDER BY execution_time DESC
LIMIT 10;
```

### Task Status

You can query task status programmatically (this would be added to the API later):

```typescript
// Get current scheduler status
const status = await schedulerManager.GetTaskStatus();
console.log(status);
```

## Timezone Considerations

- All cron expressions are evaluated in the specified timezone
- Task-specific timezones override the global scheduler timezone
- Use standard timezone names (e.g., "UTC", "America/New_York", "Europe/London")
- Invalid timezones fall back to UTC with a warning

## Security

- Schedulers run in isolated TypeScript execution environments
- Each execution gets its own clean context
- Database access is controlled by the same permissions as other endpoints
- Environment variables are sandboxed per task
- File access is limited to the configured paths

## Troubleshooting

### Task Not Running

1. Check if scheduler is enabled: `scheduler.enabled: true`
2. Verify task is enabled: `tasks[].enabled: true`
3. Check cron expression syntax
4. Verify timezone configuration
5. Check server logs for scheduler startup messages

### Execution Failures

1. Check task timeout settings
2. Review error logs for specific failure reasons
3. Test the handler function manually
4. Verify database permissions
5. Check file path resolution

### Performance Issues

1. Monitor execution duration in `scheduler_executions` table
2. Profile database queries for optimization
3. Consider breaking large tasks into smaller batches
4. Adjust timeout values if needed
5. Review memory usage patterns

## Examples

See the included example schedulers:

- `jobs-retention.ts` - Database cleanup and maintenance
- `daily-analytics.ts` - Data processing and reporting (example)

These provide templates for common scheduler patterns you can adapt for your specific needs.
