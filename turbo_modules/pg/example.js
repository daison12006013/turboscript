// Example usage of @turboscript/pg module
// This example demonstrates how to use the PostgreSQL client and pool

// Import the pg module
const pg = require('pg');

async function basicClientExample() {
    console.log('=== Basic Client Example ===');

    // Create a client with connection config
    const client = new pg.Client({
        host: 'localhost',
        port: 5432,
        database: 'testdb',
        user: 'postgres',
        password: 'password',
        sslmode: 'prefer'
    });

    try {
        // Connect to the database
        await client.connect();
        console.log('Connected to PostgreSQL');

        // Execute a simple query
        const result = await client.query('SELECT NOW() as current_time');
        console.log('Current time:', result.rows[0]);

        // Query with parameters
        const userResult = await client.query(
            'SELECT id, name, email FROM users WHERE age > $1',
            [25]
        );
        console.log(`Found ${userResult.rowCount} users`);
        console.log('Processing time:', result.processingTimeMs, 'ms');

        // Use escaping functions
        const safeName = client.escapeIdentifier('user_name');
        const safeValue = client.escapeLiteral("John's Data");
        console.log('Escaped identifier:', safeName);
        console.log('Escaped literal:', safeValue);

    } catch (error) {
        console.error('Client error:', error.message);
    } finally {
        // Always close the connection
        await client.end();
        console.log('Client connection closed');
    }
}

async function connectionPoolExample() {
    console.log('\n=== Connection Pool Example ===');

    // Create a connection pool
    const pool = new pg.Pool({
        host: 'localhost',
        port: 5432,
        database: 'testdb',
        user: 'postgres',
        password: 'password',
        max: 10,                    // Maximum pool size
        idleCount: 2,               // Idle connections to maintain
        connectionTimeoutMillis: 30000,
        queryTimeoutMillis: 5000,
        idleTimeoutMillis: 30000
    });

    try {
        // Method 1: Use pool.query() directly (recommended)
        const directResult = await pool.query('SELECT COUNT(*) as total FROM users');
        console.log('Total users (direct):', directResult.rows[0]);

        // Method 2: Checkout a client from the pool
        const client = await pool.connect();
        try {
            const clientResult = await client.query('SELECT version()');
            console.log('PostgreSQL version:', clientResult.rows[0]);
        } finally {
            // Always release the client back to the pool
            client.release();
        }

        // Pool statistics
        console.log('Pool statistics:');
        console.log('- Total connections:', pool.totalCount);
        console.log('- Idle connections:', pool.idleCount);
        console.log('- Waiting connections:', pool.waitingCount);

    } catch (error) {
        console.error('Pool error:', error.message);
    } finally {
        // Close all connections in the pool
        await pool.end();
        console.log('Pool connections closed');
    }
}

async function transactionExample() {
    console.log('\n=== Transaction Example ===');

    const client = new pg.Client({
        connectionString: 'postgresql://postgres:password@localhost:5432/testdb'
    });

    try {
        await client.connect();

        // Start transaction
        await client.query('BEGIN');

        try {
            // Execute multiple queries in transaction
            await client.query(
                'INSERT INTO users (name, email) VALUES ($1, $2)',
                ['John Doe', 'john@example.com']
            );

            await client.query(
                'INSERT INTO user_preferences (user_email, theme) VALUES ($1, $2)',
                ['john@example.com', 'dark']
            );

            // Commit transaction
            await client.query('COMMIT');
            console.log('Transaction committed successfully');

        } catch (error) {
            // Rollback on error
            await client.query('ROLLBACK');
            console.error('Transaction rolled back:', error.message);
        }

    } catch (error) {
        console.error('Transaction error:', error.message);
    } finally {
        await client.end();
    }
}

async function eventHandlingExample() {
    console.log('\n=== Event Handling Example ===');

    const client = new pg.Client({
        host: 'localhost',
        database: 'testdb',
        user: 'postgres',
        password: 'password'
    });

    // Set up event listeners
    client.on('connect', () => {
        console.log('Client connected to database');
    });

    client.on('end', () => {
        console.log('Client disconnected from database');
    });

    client.on('error', (err) => {
        console.error('Client error:', err.message);
    });

    client.on('notice', (notice) => {
        console.log('Database notice:', notice);
    });

    try {
        await client.connect();

        // Perform some operations
        const result = await client.query('SELECT 1 as test');
        console.log('Test query result:', result.rows[0]);

    } catch (error) {
        console.error('Event handling error:', error.message);
    } finally {
        await client.end();
    }
}

async function configurationOptionsExample() {
    console.log('\n=== Configuration Options Example ===');

    // Show default configuration
    console.log('Default pg configuration:', pg.defaults);

    // Show native bindings information
    console.log('Native bindings:', pg.native);

    // Advanced configuration
    const pool = new pg.Pool({
        host: 'localhost',
        port: 5432,
        database: 'testdb',
        user: 'postgres',
        password: 'password',

        // Connection pool settings
        max: 20,                    // Maximum pool size
        idleCount: 5,               // Minimum idle connections

        // Timeout settings
        connectionTimeoutMillis: 30000,    // 30 seconds
        queryTimeoutMillis: 10000,         // 10 seconds
        idleTimeoutMillis: 60000,          // 1 minute

        // PostgreSQL specific settings
        sslmode: 'require',
        application_name: 'MyApp',
        statement_timeout: 30000,
        lock_timeout: 10000,
        idle_in_transaction_session_timeout: 60000
    });

    try {
        const result = await pool.query('SELECT current_setting($1) as value', ['application_name']);
        console.log('Application name from DB:', result.rows[0]);
    } catch (error) {
        console.error('Configuration error:', error.message);
    } finally {
        await pool.end();
    }
}

// Run all examples
async function runAllExamples() {
    try {
        await basicClientExample();
        await connectionPoolExample();
        await transactionExample();
        await eventHandlingExample();
        await configurationOptionsExample();

        console.log('\n=== All examples completed ===');
    } catch (error) {
        console.error('Example error:', error.message);
    }
}

// Run examples if this file is executed directly
if (typeof module !== 'undefined' && require.main === module) {
    runAllExamples();
}

// Export for use in other modules
module.exports = {
    basicClientExample,
    connectionPoolExample,
    transactionExample,
    eventHandlingExample,
    configurationOptionsExample,
    runAllExamples
};
