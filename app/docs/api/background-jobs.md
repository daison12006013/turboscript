# Background Jobs

TurboScript provides a powerful background job system for handling asynchronous tasks like sending emails, processing data, and other time-consuming operations. This guide covers everything you need to know about implementing and managing background jobs.

## Overview

Background jobs in TurboScript allow you to:

- **Offload Heavy Tasks**: Move time-consuming operations out of request/response cycle
- **Reliable Processing**: Built-in retry mechanisms and error handling
- **Queue Management**: Configurable queue size and worker count
- **Email Integration**: Built-in email job processing with multiple drivers
- **Job Monitoring**: Track job status and history
- **Auto Cleanup**: Automatic cleanup of old job data

## Configuration

Configure background jobs in your `turboscript.yml`:

```yaml
# Background Jobs Configuration
jobs:
  enabled: true              # Enable/disable job processing
  max_workers: 5             # Number of concurrent workers
  retry_attempts: 3          # Maximum retry attempts for failed jobs
  retry_delay: 5             # Delay between retries (seconds)
  timeout: 30                # Job execution timeout (seconds)
  queue_size: 1000           # Maximum queue size
  path: ./app/queue          # Directory containing job handlers
  data_retention:
    jobs_days: 30            # Keep completed/failed jobs for 30 days
    history_days: 30         # Keep job history for 30 days
    auto_cleanup: true       # Automatically clean up old data
```

## Job Structure

Jobs are TypeScript files in the `app/queue/` directory that export an async `handle()` function:

```typescript
// app/queue/send-welcome-email.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    // Job processing logic here
    console.log('Processing job:', job);

    // Example: Send welcome email
    await turboEmail({
        to: job.email,
        subject: 'Welcome!',
        content: `Welcome ${job.name}! Thank you for joining us.`,
        driver: 'smtp'
    });
};

interface JobData {
    email: string;
    name: string;
}
```

## Dispatching Jobs

### From Route Handlers

Dispatch jobs from your API endpoints:

```typescript
// app/routes/users/create.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { name, email, password } = data;

        // Create user in database
        const hashedPassword = await hashPassword(password);
        const userResult = await turboQuery(
            'INSERT INTO users (name, email, password, created_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP) RETURNING *',
            [name, email, hashedPassword]
        );

        const user = userResult[0];

        // Dispatch background job for welcome email
        await turboJob('send-welcome-email', {
            email: user.email,
            name: user.name,
            userId: user.id
        });

        // Dispatch another job for user analytics
        await turboJob('track-user-registration', {
            userId: user.id,
            source: event.headers.referer || 'direct',
            userAgent: event.headers['user-agent']
        });

        return {
            code: 201,
            response: {
                status: "success",
                message: "User created successfully",
                data: {
                    user: {
                        id: user.id,
                        name: user.name,
                        email: user.email
                    }
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to create user"
            }
        };
    }
};
```

### Job Scheduling

Schedule jobs with delays:

```typescript
// Dispatch job with delay
await turboJob('send-reminder-email', {
    userId: user.id,
    reminderType: 'profile_completion'
}, {
    delay: 24 * 60 * 60 * 1000 // 24 hours in milliseconds
});

// Dispatch job with specific execution time
await turboJob('monthly-report', {
    userId: user.id,
    month: new Date().getMonth() + 1
}, {
    executeAt: new Date('2025-08-01T09:00:00Z')
});
```

## Common Job Examples

### Email Jobs

```typescript
// app/queue/send-confirmation-email.ts
export const handle = async (job: { email: string; token: string }, event: Event): Promise<void> => {
    const confirmationUrl = `${event.env.APP_URL}/confirm-email?token=${job.token}`;

    await turboEmail({
        to: job.email,
        subject: 'Confirm Your Email Address',
        content: `Please confirm your email by clicking: ${confirmationUrl}`,
        html: `
            <h2>Confirm Your Email</h2>
            <p>Please confirm your email address by clicking the button below:</p>
            <a href="${confirmationUrl}" style="background: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
                Confirm Email
            </a>
        `,
        driver: 'smtp'
    });
};
```

### Data Processing Jobs

```typescript
// app/queue/process-user-data.ts
export const handle = async (job: { userId: string; dataType: string }, event: Event): Promise<void> => {
    const { userId, dataType } = job;

    try {
        // Fetch user data
        const userData = await turboQuery(
            'SELECT * FROM users WHERE id = $1',
            [userId]
        );

        if (userData.length === 0) {
            throw new Error(`User not found: ${userId}`);
        }

        const user = userData[0];

        // Process based on data type
        switch (dataType) {
            case 'analytics':
                await processUserAnalytics(user);
                break;
            case 'recommendations':
                await generateRecommendations(user);
                break;
            case 'cleanup':
                await cleanupUserData(user);
                break;
            default:
                throw new Error(`Unknown data type: ${dataType}`);
        }

        // Update processing status
        await turboQuery(
            'UPDATE users SET last_processed_at = CURRENT_TIMESTAMP WHERE id = $1',
            [userId]
        );

    } catch (error) {
        console.error(`Failed to process user data: ${error.message}`);
        throw error; // This will trigger retry mechanism
    }
};

async function processUserAnalytics(user: any): Promise<void> {
    // Calculate user engagement metrics
    const engagementData = await turboQuery(`
        SELECT
            COUNT(*) as login_count,
            MAX(created_at) as last_login,
            AVG(session_duration) as avg_session_duration
        FROM user_sessions
        WHERE user_id = $1 AND created_at > NOW() - INTERVAL '30 days'
    `, [user.id]);

    // Store analytics
    await turboQuery(
        'INSERT INTO user_analytics (user_id, login_count, last_login, avg_session_duration, calculated_at) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)',
        [user.id, engagementData[0].login_count, engagementData[0].last_login, engagementData[0].avg_session_duration]
    );
}

async function generateRecommendations(user: any): Promise<void> {
    // Generate personalized recommendations
    const userPreferences = await turboQuery(
        'SELECT preferences FROM user_preferences WHERE user_id = $1',
        [user.id]
    );

    // Complex recommendation logic here...
    const recommendations = await calculateRecommendations(user, userPreferences[0]);

    // Store recommendations
    await turboQuery(
        'INSERT INTO user_recommendations (user_id, recommendations, generated_at) VALUES ($1, $2, CURRENT_TIMESTAMP)',
        [user.id, JSON.stringify(recommendations)]
    );
}

async function cleanupUserData(user: any): Promise<void> {
    // Clean up old user data
    await turboQuery(
        'DELETE FROM user_sessions WHERE user_id = $1 AND created_at < NOW() - INTERVAL \'90 days\'',
        [user.id]
    );

    await turboQuery(
        'DELETE FROM user_activities WHERE user_id = $1 AND created_at < NOW() - INTERVAL \'180 days\'',
        [user.id]
    );
}
```

### File Processing Jobs

```typescript
// app/queue/process-file-upload.ts
export const handle = async (job: { fileId: string; userId: string; operation: string }, event: Event): Promise<void> => {
    const { fileId, userId, operation } = job;

    try {
        // Fetch file metadata
        const fileData = await turboQuery(
            'SELECT * FROM file_uploads WHERE id = $1 AND user_id = $2',
            [fileId, userId]
        );

        if (fileData.length === 0) {
            throw new Error(`File not found: ${fileId}`);
        }

        const file = fileData[0];

        // Update status to processing
        await turboQuery(
            'UPDATE file_uploads SET status = $1, processing_started_at = CURRENT_TIMESTAMP WHERE id = $2',
            ['processing', fileId]
        );

        // Process based on operation type
        switch (operation) {
            case 'image_resize':
                await processImageResize(file);
                break;
            case 'document_parse':
                await processDocumentParse(file);
                break;
            case 'virus_scan':
                await processVirusScan(file);
                break;
            default:
                throw new Error(`Unknown operation: ${operation}`);
        }

        // Update status to completed
        await turboQuery(
            'UPDATE file_uploads SET status = $1, processing_completed_at = CURRENT_TIMESTAMP WHERE id = $2',
            ['completed', fileId]
        );

        // Notify user of completion
        await turboJob('send-processing-complete-notification', {
            userId,
            fileId,
            fileName: file.original_name
        });

    } catch (error) {
        // Update status to failed
        await turboQuery(
            'UPDATE file_uploads SET status = $1, error_message = $2 WHERE id = $3',
            ['failed', error.message, fileId]
        );

        console.error(`File processing failed: ${error.message}`);
        throw error;
    }
};

async function processImageResize(file: any): Promise<void> {
    // Image resizing logic
    console.log(`Resizing image: ${file.file_path}`);
    // Implementation would use image processing library
}

async function processDocumentParse(file: any): Promise<void> {
    // Document parsing logic
    console.log(`Parsing document: ${file.file_path}`);
    // Implementation would use document parsing library
}

async function processVirusScan(file: any): Promise<void> {
    // Virus scanning logic
    console.log(`Scanning file: ${file.file_path}`);
    // Implementation would use antivirus API
}
```

### Periodic Jobs

```typescript
// app/queue/daily-cleanup.ts
export const handle = async (job: { cleanup_type: string }, event: Event): Promise<void> => {
    const { cleanup_type } = job;

    console.log(`Starting daily cleanup: ${cleanup_type}`);

    try {
        switch (cleanup_type) {
            case 'expired_sessions':
                await cleanupExpiredSessions();
                break;
            case 'old_logs':
                await cleanupOldLogs();
                break;
            case 'temp_files':
                await cleanupTempFiles();
                break;
            case 'all':
                await cleanupExpiredSessions();
                await cleanupOldLogs();
                await cleanupTempFiles();
                break;
            default:
                throw new Error(`Unknown cleanup type: ${cleanup_type}`);
        }

        console.log(`Daily cleanup completed: ${cleanup_type}`);

    } catch (error) {
        console.error(`Daily cleanup failed: ${error.message}`);
        throw error;
    }
};

async function cleanupExpiredSessions(): Promise<void> {
    const result = await turboQuery(
        'DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP'
    );
    console.log(`Cleaned up ${result.rowsAffected} expired sessions`);
}

async function cleanupOldLogs(): Promise<void> {
    const result = await turboQuery(
        'DELETE FROM application_logs WHERE created_at < NOW() - INTERVAL \'30 days\''
    );
    console.log(`Cleaned up ${result.rowsAffected} old log entries`);
}

async function cleanupTempFiles(): Promise<void> {
    const result = await turboQuery(
        'DELETE FROM temp_files WHERE created_at < NOW() - INTERVAL \'24 hours\''
    );
    console.log(`Cleaned up ${result.rowsAffected} temporary files`);
}
```

## Error Handling and Retries

Jobs automatically retry on failure based on your configuration:

```typescript
// app/queue/unreliable-external-api.ts
export const handle = async (job: { userId: string; apiData: any }, event: Event): Promise<void> => {
    const { userId, apiData } = job;

    try {
        // Simulate external API call that might fail
        const response = await fetch('https://external-api.example.com/process', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${event.env.EXTERNAL_API_KEY}`
            },
            body: JSON.stringify(apiData)
        });

        if (!response.ok) {
            throw new Error(`External API failed: ${response.status} ${response.statusText}`);
        }

        const result = await response.json();

        // Store successful result
        await turboQuery(
            'INSERT INTO external_api_results (user_id, result_data, processed_at) VALUES ($1, $2, CURRENT_TIMESTAMP)',
            [userId, JSON.stringify(result)]
        );

        console.log(`Successfully processed external API request for user ${userId}`);

    } catch (error) {
        console.error(`External API processing failed: ${error.message}`);

        // Log the failure
        await turboQuery(
            'INSERT INTO job_failures (job_name, user_id, error_message, failed_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP)',
            ['unreliable-external-api', userId, error.message]
        );

        // Re-throw to trigger retry mechanism
        throw error;
    }
};
```

## Job Monitoring

### Check Job Status

```typescript
// app/routes/jobs/status.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const jobId = event.pathParameters.jobId;

        // In a real implementation, you would query job status from the database
        // This is a conceptual example since job tracking would need to be implemented
        const jobStatus = await turboQuery(
            'SELECT * FROM job_queue WHERE id = $1',
            [jobId]
        );

        if (jobStatus.length === 0) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Job not found"
                }
            };
        }

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    job: jobStatus[0]
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to get job status"
            }
        };
    }
};
```

### List Jobs

```typescript
// app/routes/jobs/list.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const status = event.queryParameters.status || 'all';
        const page = parseInt(event.queryParameters.page || '1');
        const limit = parseInt(event.queryParameters.limit || '50');
        const offset = (page - 1) * limit;

        let whereClause = '';
        let params = [limit, offset];

        if (status !== 'all') {
            whereClause = 'WHERE status = $3';
            params.push(status);
        }

        const jobs = await turboQuery(
            `SELECT * FROM job_queue ${whereClause} ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
            params
        );

        const totalCount = await turboQuery(
            `SELECT COUNT(*) as count FROM job_queue ${whereClause}`,
            status !== 'all' ? [status] : []
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    jobs,
                    pagination: {
                        page,
                        limit,
                        total: totalCount[0].count,
                        pages: Math.ceil(totalCount[0].count / limit)
                    }
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to list jobs"
            }
        };
    }
};
```

## Advanced Job Patterns

### Job with Database Operations

```typescript
// app/queue/send-confirmation-email.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        const { userId, email, confirmationToken } = job;

        // Update user confirmation status
        await turboQuery(
            'UPDATE users SET email_confirmation_sent_at = NOW() WHERE id = $1',
            [userId]
        );

        // Send confirmation email
        await turboEmail({
            to: email,
            subject: 'Confirm Your Email Address',
            content: `
                <h2>Welcome!</h2>
                <p>Please click the link below to confirm your email address:</p>
                <a href="https://yourapp.com/confirm?token=${confirmationToken}">Confirm Email</a>
                <p>This link will expire in 24 hours.</p>
            `,
            html: true,
            driver: 'smtp'
        });

        // Log email sent for tracking
        await turboQuery(
            'INSERT INTO email_log (user_id, email_type, sent_at, status) VALUES ($1, $2, NOW(), $3)',
            [userId, 'confirmation', 'sent']
        );

        console.log(`Confirmation email sent to ${email}`);
    } catch (error) {
        console.error('Failed to send confirmation email:', error);

        // Log failure for debugging
        await turboQuery(
            'INSERT INTO email_log (user_id, email_type, sent_at, status, error_message) VALUES ($1, $2, NOW(), $3, $4)',
            [job.userId, 'confirmation', 'failed', error.message]
        );

        throw error; // Re-throw to trigger retry mechanism
    }
};

interface JobData {
    userId: string;
    email: string;
    confirmationToken: string;
}
```

### Batch Processing Job

```typescript
// app/queue/process-daily-analytics.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        const { date } = job;
        console.log(`Processing analytics for ${date}`);

        // Execute multiple analytics queries in parallel
        const [
            userStats,
            orderStats,
            pageViews,
            conversionRates
        ] = await Promise.all([
            turboQuery(
                'SELECT COUNT(*) as new_users FROM users WHERE DATE(created_at) = $1',
                [date]
            ),
            turboQuery(
                'SELECT COUNT(*) as orders, SUM(total) as revenue FROM orders WHERE DATE(created_at) = $1',
                [date]
            ),
            turboQuery(
                'SELECT page, COUNT(*) as views FROM page_analytics WHERE DATE(created_at) = $1 GROUP BY page',
                [date]
            ),
            turboQuery(
                'SELECT COUNT(*) as conversions FROM conversions WHERE DATE(created_at) = $1',
                [date]
            )
        ]);

        // Store aggregated analytics
        await turboQuery(
            'INSERT INTO daily_analytics (date, new_users, orders, revenue, conversions, page_views, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())',
            [
                date,
                userStats[0].new_users,
                orderStats[0].orders,
                orderStats[0].revenue || 0,
                conversionRates[0].conversions,
                JSON.stringify(pageViews)
            ]
        );

        // Clean up raw analytics data older than 30 days
        await turboQuery(
            'DELETE FROM page_analytics WHERE created_at < NOW() - INTERVAL \'30 days\''
        );

        console.log(`Analytics processing completed for ${date}`);
    } catch (error) {
        console.error('Analytics processing failed:', error);
        throw error;
    }
};

interface JobData {
    date: string; // YYYY-MM-DD format
}
```

### Multi-Step Job Processing

```typescript
// app/queue/process-order-fulfillment.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        const { orderId, userId } = job;

        // Step 1: Validate order
        const orders = await turboQuery(
            'SELECT * FROM orders WHERE id = $1 AND status = $2',
            [orderId, 'pending']
        );

        if (!orders.length) {
            throw new Error(`Order ${orderId} not found or not pending`);
        }

        const order = orders[0];

        // Step 2: Process payment
        await turboQuery(
            'UPDATE orders SET status = $1, processing_started_at = NOW() WHERE id = $2',
            ['processing', orderId]
        );

        // Step 3: Update inventory (parallel operations)
        const orderItems = await turboQuery(
            'SELECT product_id, quantity FROM order_items WHERE order_id = $1',
            [orderId]
        );

        const inventoryUpdates = orderItems.map(item =>
            turboQuery(
                'UPDATE products SET stock = stock - $1 WHERE id = $2',
                [item.quantity, item.product_id]
            )
        );

        await Promise.all(inventoryUpdates);

        // Step 4: Generate shipping label and notify warehouse
        const [shippingResult, warehouseNotification] = await Promise.all([
            turboQuery(
                'INSERT INTO shipping_labels (order_id, tracking_number, created_at) VALUES ($1, $2, NOW()) RETURNING *',
                [orderId, `TRK${Date.now()}`]
            ),
            // Dispatch another job for warehouse notification
            turboJob('notify-warehouse', {
                orderId,
                priority: order.priority || 'normal',
                items: orderItems
            })
        ]);

        // Step 5: Send customer notification
        await turboJob('send-order-confirmation', {
            userId,
            orderId,
            trackingNumber: shippingResult[0].tracking_number
        });

        // Step 6: Update order status
        await turboQuery(
            'UPDATE orders SET status = $1, tracking_number = $2, fulfilled_at = NOW() WHERE id = $3',
            ['fulfilled', shippingResult[0].tracking_number, orderId]
        );

        console.log(`Order ${orderId} successfully processed`);
    } catch (error) {
        // Update order status to failed
        await turboQuery(
            'UPDATE orders SET status = $1, error_message = $2 WHERE id = $3',
            ['failed', error.message, job.orderId]
        );

        console.error(`Order processing failed for ${job.orderId}:`, error);
        throw error;
    }
};

interface JobData {
    orderId: string;
    userId: string;
}
```

## Job Monitoring and Management

### Job Status Tracking

TurboScript automatically tracks job execution:

```typescript
// Check job status in your route handlers
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { jobId } = event.pathParameters;

        const jobInfo = await turboQuery(
            'SELECT * FROM jobs WHERE id = $1',
            [jobId]
        );

        if (!jobInfo.length) {
            return {
                code: 404,
                response: {
                    status: "error",
                    message: "Job not found"
                }
            };
        }

        const job = jobInfo[0];

        // Get job history for detailed status
        const jobHistory = await turboQuery(
            'SELECT * FROM job_history WHERE job_id = $1 ORDER BY created_at DESC',
            [jobId]
        );

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    job: {
                        id: job.id,
                        status: job.status,
                        created_at: job.created_at,
                        completed_at: job.completed_at,
                        error_message: job.error_message,
                        retry_count: job.retry_count
                    },
                    history: jobHistory
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch job status"
            }
        };
    }
};
```

### Bulk Job Operations

```typescript
// app/routes/admin/jobs/bulk-retry.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const { jobIds } = event.body as { jobIds: string[] };

        if (!jobIds || !Array.isArray(jobIds)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "jobIds array is required"
                }
            };
        }

        // Reset failed jobs for retry
        const retryResult = await turboQuery(
            'UPDATE jobs SET status = $1, retry_count = retry_count + 1, error_message = NULL WHERE id = ANY($2) AND status IN ($3, $4)',
            ['pending', jobIds, 'failed', 'timeout']
        );

        return {
            code: 200,
            response: {
                status: "success",
                message: `${retryResult.rowsAffected} jobs queued for retry`,
                data: {
                    jobs_retried: retryResult.rowsAffected
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to retry jobs"
            }
        };
    }
};
```

## Email Integration

TurboScript provides built-in email capabilities for jobs:

### SMTP Configuration

```yaml
# turboscript.yml
email:
  default_driver: smtp
  drivers:
    smtp:
      host: smtp.gmail.com
      port: 587
      username: your-email@gmail.com
      password: your-app-password
      encryption: tls
      from_address: noreply@yourapp.com
      from_name: "Your App Name"
```

### Advanced Email Job

```typescript
// app/queue/send-newsletter.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        const { newsletterId, segmentId } = job;

        // Get newsletter content
        const newsletters = await turboQuery(
            'SELECT * FROM newsletters WHERE id = $1',
            [newsletterId]
        );

        if (!newsletters.length) {
            throw new Error(`Newsletter ${newsletterId} not found`);
        }

        const newsletter = newsletters[0];

        // Get subscriber segment
        const subscribers = await turboQuery(
            'SELECT u.email, u.name, u.id FROM users u INNER JOIN user_segments us ON u.id = us.user_id WHERE us.segment_id = $1 AND u.subscribed = true',
            [segmentId]
        );

        console.log(`Sending newsletter to ${subscribers.length} subscribers`);

        // Send emails in batches to avoid overwhelming the SMTP server
        const batchSize = 50;
        for (let i = 0; i < subscribers.length; i += batchSize) {
            const batch = subscribers.slice(i, i + batchSize);

            const emailPromises = batch.map(async (subscriber) => {
                try {
                    // Personalize content
                    const personalizedContent = newsletter.content
                        .replace('{{name}}', subscriber.name)
                        .replace('{{unsubscribe_link}}', `https://yourapp.com/unsubscribe?token=${subscriber.id}`);

                    await turboEmail({
                        to: subscriber.email,
                        subject: newsletter.subject,
                        content: personalizedContent,
                        html: true,
                        driver: 'smtp'
                    });

                    // Track delivery
                    await turboQuery(
                        'INSERT INTO email_deliveries (newsletter_id, user_id, sent_at, status) VALUES ($1, $2, NOW(), $3)',
                        [newsletterId, subscriber.id, 'sent']
                    );

                } catch (error) {
                    console.error(`Failed to send to ${subscriber.email}:`, error);

                    // Track failure
                    await turboQuery(
                        'INSERT INTO email_deliveries (newsletter_id, user_id, sent_at, status, error_message) VALUES ($1, $2, NOW(), $3, $4)',
                        [newsletterId, subscriber.id, 'failed', error.message]
                    );
                }
            });

            await Promise.all(emailPromises);

            // Add delay between batches to avoid rate limiting
            if (i + batchSize < subscribers.length) {
                await new Promise(resolve => setTimeout(resolve, 1000)); // 1 second delay
            }
        }

        // Update newsletter status
        await turboQuery(
            'UPDATE newsletters SET status = $1, sent_at = NOW(), recipients_count = $2 WHERE id = $3',
            ['sent', subscribers.length, newsletterId]
        );

        console.log(`Newsletter ${newsletterId} sent successfully to ${subscribers.length} subscribers`);
    } catch (error) {
        console.error('Newsletter sending failed:', error);

        // Update newsletter status to failed
        await turboQuery(
            'UPDATE newsletters SET status = $1, error_message = $2 WHERE id = $3',
            ['failed', error.message, job.newsletterId]
        );

        throw error;
    }
};

interface JobData {
    newsletterId: string;
    segmentId: string;
}
```

## Error Handling and Retry Logic

### Automatic Retry Configuration

Configure retry behavior in `turboscript.yml`:

```yaml
jobs:
  retry_attempts: 3          # Maximum retry attempts
  retry_delay: 5             # Initial delay between retries (seconds)
  retry_backoff: exponential # exponential, linear, or fixed
  max_retry_delay: 300       # Maximum delay (5 minutes)
```

### Custom Error Handling

```typescript
// app/queue/process-payment.ts
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        const { orderId, paymentMethodId, amount } = job;

        // Attempt payment processing
        const paymentResult = await processPayment(paymentMethodId, amount);

        if (!paymentResult.success) {
            // Determine if error is retryable
            if (paymentResult.error_code === 'temporary_failure') {
                throw new Error(`Payment temporarily failed: ${paymentResult.message}`);
            } else {
                // Permanent failure - don't retry
                await turboQuery(
                    'UPDATE orders SET status = $1, payment_error = $2 WHERE id = $3',
                    ['payment_failed', paymentResult.message, orderId]
                );

                // Send notification to customer
                await turboJob('send-payment-failed-notification', {
                    orderId,
                    errorMessage: paymentResult.message
                });

                return; // Exit without throwing to prevent retry
            }
        }

        // Payment successful
        await turboQuery(
            'UPDATE orders SET status = $1, payment_id = $2, paid_at = NOW() WHERE id = $3',
            ['paid', paymentResult.payment_id, orderId]
        );

        // Trigger fulfillment
        await turboJob('process-order-fulfillment', { orderId });

    } catch (error) {
        console.error(`Payment processing failed for order ${job.orderId}:`, error);
        throw error; // Will trigger retry if within retry limits
    }
};

interface JobData {
    orderId: string;
    paymentMethodId: string;
    amount: number;
}
```

## Performance and Scaling

### Worker Configuration

Scale job processing based on your needs:

```yaml
# Development
jobs:
  max_workers: 2
  queue_size: 100

# Production
jobs:
  max_workers: 10
  queue_size: 5000
  worker_timeout: 300  # 5 minutes for complex jobs
```

### Job Prioritization

```typescript
// High priority job dispatch
await turboJob('send-critical-alert', {
    userId,
    alertType: 'security_breach',
    priority: 'high'  // Jobs with higher priority are processed first
});

// Normal priority job
await turboJob('send-welcome-email', {
    userId,
    priority: 'normal'
});
```

### Monitoring Performance

```typescript
// app/routes/admin/jobs/stats.ts
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const [
            queueStats,
            processingStats,
            recentErrors
        ] = await Promise.all([
            turboQuery(
                'SELECT status, COUNT(*) as count FROM jobs GROUP BY status'
            ),
            turboQuery(
                'SELECT AVG(EXTRACT(EPOCH FROM (completed_at - created_at))) as avg_processing_time FROM jobs WHERE status = $1 AND completed_at > NOW() - INTERVAL \'24 hours\'',
                ['completed']
            ),
            turboQuery(
                'SELECT job_path, error_message, COUNT(*) as error_count FROM jobs WHERE status = $1 AND created_at > NOW() - INTERVAL \'1 hour\' GROUP BY job_path, error_message ORDER BY error_count DESC LIMIT 10',
                ['failed']
            )
        ]);

        return {
            code: 200,
            response: {
                status: "success",
                data: {
                    queue_stats: queueStats,
                    avg_processing_time_seconds: processingStats[0]?.avg_processing_time || 0,
                    recent_errors: recentErrors
                }
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Failed to fetch job statistics"
            }
        };
    }
};
```

## Testing Background Jobs

### Unit Testing Jobs

```typescript
// test/queue/send-welcome-email.test.ts
import { handle } from '../../app/queue/send-welcome-email';

describe('Send Welcome Email Job', () => {
    const mockEvent = {
        headers: {},
        queryParameters: {},
        pathParameters: {},
        body: {},
        env: {}
    };

    test('should send welcome email successfully', async () => {
        const jobData = {
            email: 'test@example.com',
            name: 'Test User',
            userId: '123'
        };

        // Mock database and email functions
        jest.mock('../../app/utils/database', () => ({
            turboQuery: jest.fn().mockResolvedValue([])
        }));

        jest.mock('../../app/utils/email', () => ({
            turboEmail: jest.fn().mockResolvedValue({})
        }));

        await expect(handle(jobData, mockEvent)).resolves.not.toThrow();
    });
});
```

### Integration Testing

```bash
# Dispatch a test job
curl -X POST http://localhost:7890/test/dispatch-job \
  -H "Content-Type: application/json" \
  -d '{
    "jobPath": "send-welcome-email",
    "payload": {
      "email": "test@example.com",
      "name": "Test User"
    }
  }'

# Check job status
curl http://localhost:7890/admin/jobs/12345
```

## Best Practices

### Job Design Principles

1. **Idempotent Operations**: Jobs should be safe to run multiple times
2. **Atomic Operations**: Keep jobs focused on single responsibilities
3. **Graceful Degradation**: Handle partial failures appropriately
4. **Proper Logging**: Include comprehensive logging for debugging
5. **Resource Management**: Clean up resources and avoid memory leaks

### Error Handling Guidelines

```typescript
export const handle = async (job: JobData, event: Event): Promise<void> => {
    try {
        // Job logic here

    } catch (error) {
        // Log error with context
        console.error('Job failed:', {
            jobPath: 'your-job-name',
            jobData: job,
            error: error.message,
            stack: error.stack
        });

        // Determine if error is retryable
        if (isRetryableError(error)) {
            throw error; // Will trigger retry
        } else {
            // Handle permanent failure
            await handlePermanentFailure(job, error);
            return; // Don't throw to prevent retry
        }
    }
};

function isRetryableError(error: Error): boolean {
    // Network errors, temporary service unavailable, etc.
    return error.message.includes('ECONNRESET') ||
           error.message.includes('timeout') ||
           error.message.includes('503');
}
```

### Security Considerations

- **Input Validation**: Always validate job payload data
- **Access Control**: Ensure jobs only access authorized resources
- **Sensitive Data**: Never log sensitive information
- **Resource Limits**: Set appropriate timeouts and memory limits

---

## Navigation

**Previous:** [← Authentication](api/authentication.md)
**Next:** [API Examples →](api/examples.md)

## Related Topics

- [Database Operations](api/database-operations.md)
- [Route Handler API](api/route-handlers.md)
- [Performance Optimization](guides/performance.md)
- [Security Guidelines](guides/security.md)
