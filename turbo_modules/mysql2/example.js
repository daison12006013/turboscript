// Example usage of @turboscript/mysql2 module
// This example demonstrates how to use the MySQL2 client, pool, and promise-based APIs

// Import the mysql2 module
const mysql2 = require('mysql2');

async function basicConnectionExample() {
    console.log('=== Basic Connection Example (Callback-based) ===');

    // Create a connection
    const connection = mysql2.createConnection({
        host: 'localhost',
        port: 3306,
        user: 'root',
        password: 'password',
        database: 'testdb',
        charset: 'utf8mb4',
        timezone: '+00:00'
    });

    // Connect to the database
    connection.connect((err) => {
        if (err) {
            console.error('Connection failed:', err.message);
            return;
        }
        console.log('Connected to MySQL');

        // Execute a simple query
        connection.query('SELECT NOW() as current_time', (err, results, fields) => {
            if (err) {
                console.error('Query failed:', err.message);
                return;
            }
            console.log('Current time:', results[0]);
            console.log('Field info:', fields[0]);

            // Query with parameters
            connection.query(
                'SELECT id, name, email FROM users WHERE age > ?',
                [25],
                (err, results, fields) => {
                    if (err) {
                        console.error('Parameterized query failed:', err.message);
                        return;
                    }
                    console.log(`Found ${results.length} users`);
                    console.log('Results:', results);

                    // Close the connection
                    connection.end((err) => {
                        if (err) {
                            console.error('Error closing connection:', err.message);
                        } else {
                            console.log('Connection closed');
                        }
                    });
                }
            );
        });
    });
}

async function connectionPoolExample() {
    console.log('\n=== Connection Pool Example (Callback-based) ===');

    // Create a connection pool
    const pool = mysql2.createPool({
        host: 'localhost',
        port: 3306,
        user: 'root',
        password: 'password',
        database: 'testdb',
        connectionLimit: 10,        // Maximum pool size
        queueLimit: 0,              // Unlimited queue
        acquireTimeout: 60000,      // 60 seconds
        timeout: 60000,             // 60 seconds
        reconnect: true,
        charset: 'utf8mb4'
    });

    // Method 1: Use pool.query() directly (recommended)
    pool.query('SELECT COUNT(*) as total FROM users', (err, results, fields) => {
        if (err) {
            console.error('Pool query failed:', err.message);
            return;
        }
        console.log('Total users (direct):', results[0]);
    });

    // Method 2: Get a connection from the pool
    pool.getConnection((err, connection) => {
        if (err) {
            console.error('Pool connection failed:', err.message);
            return;
        }
        console.log('Got connection from pool');

        // Use the connection
        connection.query('SELECT version() as version', (err, results, fields) => {
            if (err) {
                console.error('Pool connection query failed:', err.message);
            } else {
                console.log('MySQL version:', results[0]);
            }

            // Always release the connection back to the pool
            connection.release();
            console.log('Connection released back to pool');
        });
    });

    // Close the pool after some time
    setTimeout(() => {
        pool.end((err) => {
            if (err) {
                console.error('Error closing pool:', err.message);
            } else {
                console.log('Pool closed');
            }
        });
    }, 5000);
}

async function promiseBasedExample() {
    console.log('\n=== Promise-based API Example ===');

    // Create a promise-based connection
    const connection = mysql2.promise.createConnection({
        host: 'localhost',
        port: 3306,
        user: 'root',
        password: 'password',
        database: 'testdb',
        charset: 'utf8mb4'
    });

    try {
        // Connect to the database
        await connection.connect();
        console.log('Connected to MySQL (Promise)');

        // Execute a simple query
        const [rows, fields] = await connection.query('SELECT NOW() as current_time');
        console.log('Current time (Promise):', rows[0]);

        // Query with parameters
        const [userRows] = await connection.query(
            'SELECT id, name, email FROM users WHERE age > ?',
            [25]
        );
        console.log(`Found ${userRows.length} users (Promise)`);

        // Use execute for prepared statements
        const [executeRows] = await connection.execute(
            'SELECT * FROM users WHERE id = ? AND status = ?',
            [1, 'active']
        );
        console.log('Execute results:', executeRows);

    } catch (error) {
        console.error('Promise connection error:', error.message);
    } finally {
        // Always close the connection
        await connection.end();
        console.log('Promise connection closed');
    }
}

async function promisePoolExample() {
    console.log('\n=== Promise Pool Example ===');

    // Create a promise-based pool
    const pool = mysql2.promise.createPool({
        host: 'localhost',
        port: 3306,
        user: 'root',
        password: 'password',
        database: 'testdb',
        connectionLimit: 10,
        acquireTimeout: 60000,
        charset: 'utf8mb4'
    });

    try {
        // Method 1: Use pool.query() directly
        const [directRows] = await pool.query('SELECT COUNT(*) as total FROM users');
        console.log('Total users (Promise Pool):', directRows[0]);

        // Method 2: Get a connection from the pool
        const connection = await pool.getConnection();
        try {
            const [rows] = await connection.query('SELECT DATABASE() as current_db');
            console.log('Current database:', rows[0]);
        } finally {
            // Always release the connection
            connection.release();
            console.log('Promise pool connection released');
        }

        // Execute multiple queries in parallel
        const [users, products, orders] = await Promise.all([
            pool.query('SELECT COUNT(*) as count FROM users'),
            pool.query('SELECT COUNT(*) as count FROM products'),
            pool.query('SELECT COUNT(*) as count FROM orders')
        ]);

        console.log('Parallel query results:');
        console.log('- Users:', users[0][0]);
        console.log('- Products:', products[0][0]);
        console.log('- Orders:', orders[0][0]);

    } catch (error) {
        console.error('Promise pool error:', error.message);
    } finally {
        // Close the pool
        await pool.end();
        console.log('Promise pool closed');
    }
}

async function transactionExample() {
    console.log('\n=== Transaction Example ===');

    const connection = mysql2.promise.createConnection({
        host: 'localhost',
        user: 'root',
        password: 'password',
        database: 'testdb'
    });

    try {
        await connection.connect();

        // Start transaction
        await connection.beginTransaction();
        console.log('Transaction started');

        try {
            // Execute multiple queries in transaction
            await connection.execute(
                'INSERT INTO users (name, email) VALUES (?, ?)',
                ['John Doe', 'john@example.com']
            );

            await connection.execute(
                'INSERT INTO user_preferences (user_email, theme) VALUES (?, ?)',
                ['john@example.com', 'dark']
            );

            // Commit transaction
            await connection.commit();
            console.log('Transaction committed successfully');

        } catch (error) {
            // Rollback on error
            await connection.rollback();
            console.error('Transaction rolled back:', error.message);
        }

    } catch (error) {
        console.error('Transaction error:', error.message);
    } finally {
        await connection.end();
    }
}

async function utilityFunctionsExample() {
    console.log('\n=== Utility Functions Example ===');

    // Test escape functions
    const userInput = "John's Data";
    const tableName = "user_table";

    console.log('Original value:', userInput);
    console.log('Escaped value:', mysql2.escape(userInput));
    console.log('Original identifier:', tableName);
    console.log('Escaped identifier:', mysql2.escapeId(tableName));

    // Test format function
    const sql = 'SELECT * FROM users WHERE name = ? AND age > ?';
    const values = ['John', 25];
    const formatted = mysql2.format(sql, values);
    console.log('Formatted query:', formatted);

    // Test raw function (for including raw SQL)
    const rawValue = mysql2.raw('NOW()');
    console.log('Raw value:', rawValue);
}

async function advancedConfigurationExample() {
    console.log('\n=== Advanced Configuration Example ===');

    const pool = mysql2.createPool({
        host: 'localhost',
        port: 3306,
        user: 'root',
        password: 'password',
        database: 'testdb',

        // Connection settings
        charset: 'utf8mb4',
        timezone: '+00:00',

        // Pool settings
        connectionLimit: 20,
        queueLimit: 50,
        acquireTimeout: 60000,

        // Timeout settings
        timeout: 60000,
        readTimeout: 30000,
        writeTimeout: 30000,

        // Security settings
        ssl: false,
        insecureAuth: false,

        // Data processing settings
        multipleStatements: false,
        dateStrings: false,
        supportBigNumbers: true,
        bigNumberStrings: false,
        stringifyObjects: false,

        // Performance settings
        enableKeepAlive: true,
        keepAliveInitialDelay: 0,

        // Debug settings
        debug: false,
        trace: false,

        // Type casting
        typeCast: true
    });

    pool.query('SELECT 1 as test', (err, results, fields) => {
        if (err) {
            console.error('Advanced config query failed:', err.message);
        } else {
            console.log('Advanced config test successful:', results[0]);
        }

        pool.end();
    });
}

async function errorHandlingExample() {
    console.log('\n=== Error Handling Example ===');

    const connection = mysql2.promise.createConnection({
        host: 'localhost',
        user: 'root',
        password: 'wrong_password',  // Intentional error
        database: 'testdb'
    });

    try {
        await connection.connect();
    } catch (error) {
        console.error('Expected connection error:', error.message);
    }

    // Test with correct credentials but wrong query
    const correctConnection = mysql2.promise.createConnection({
        host: 'localhost',
        user: 'root',
        password: 'password',
        database: 'testdb'
    });

    try {
        await correctConnection.connect();

        // Intentional SQL error
        await correctConnection.query('SELECT * FROM nonexistent_table');
    } catch (error) {
        console.error('Expected query error:', error.message);
    } finally {
        try {
            await correctConnection.end();
        } catch (e) {
            // Ignore close errors
        }
    }
}

// Run all examples
async function runAllExamples() {
    try {
        await basicConnectionExample();
        await connectionPoolExample();
        await promiseBasedExample();
        await promisePoolExample();
        await transactionExample();
        await utilityFunctionsExample();
        await advancedConfigurationExample();
        await errorHandlingExample();

        console.log('\n=== All MySQL2 examples completed ===');
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
    basicConnectionExample,
    connectionPoolExample,
    promiseBasedExample,
    promisePoolExample,
    transactionExample,
    utilityFunctionsExample,
    advancedConfigurationExample,
    errorHandlingExample,
    runAllExamples
};
