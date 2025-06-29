// TypeScript definitions for @turboscript/mysql2
// Compatible with Node.js mysql2 library

export interface ConnectionOptions {
    host?: string;
    port?: number;
    user?: string;
    password?: string;
    database?: string;
    charset?: string;
    timezone?: string;
    timeout?: number;
    readTimeout?: number;
    writeTimeout?: number;
    acquireTimeout?: number;
    connectionLimit?: number;
    queueLimit?: number;
    ssl?: string | boolean | object;
    multipleStatements?: boolean;
    dateStrings?: boolean;
    debug?: boolean;
    trace?: boolean;
    supportBigNumbers?: boolean;
    bigNumberStrings?: boolean;
    insecureAuth?: boolean;
    typeCast?: boolean | ((field: any, next: () => void) => any);
    stringifyObjects?: boolean;
    enableKeepAlive?: boolean;
    keepAliveInitialDelay?: number;
}

export interface QueryOptions {
    sql: string;
    values?: any[];
    timeout?: number;
    typeCast?: boolean | ((field: any, next: () => void) => any);
    nestTables?: boolean | string;
    rowsAsArray?: boolean;
}

export interface FieldInfo {
    name: string;
    type: string;
    length: number;
    decimals: number;
    flags: number;
    default: string;
    zerofill: boolean;
    protocol41: boolean;
    charsetNr: number;
    database: string;
    table: string;
    orgTable: string;
    orgName: string;
}

export interface QueryResult {
    results: any[];
    fields: FieldInfo[];
    affectedRows: number;
    insertId: number;
    changedRows: number;
    serverStatus: number;
    warningCount: number;
    message: string;
}

export interface ConnectionStats {
    Uptime: number;
    [key: string]: any;
}

// Callback-based interfaces
export interface Connection {
    connect(callback?: (err: Error | null) => void): void;
    end(callback?: (err: Error | null) => void): void;
    destroy(): void;

    query(sql: string, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    query(sql: string, values: any[], callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    query(options: QueryOptions, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;

    execute(sql: string, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    execute(sql: string, values: any[], callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    execute(options: QueryOptions, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;

    beginTransaction(callback: (err: Error | null) => void): void;
    commit(callback: (err: Error | null) => void): void;
    rollback(callback: (err: Error | null) => void): void;

    changeUser(options: any, callback: (err: Error | null) => void): void;
    ping(callback: (err: Error | null) => void): void;
    statistics(callback: (err: Error | null, stats?: ConnectionStats) => void): void;

    format(sql: string, values?: any[]): string;
    escape(value: any): string;
    escapeId(identifier: string): string;

    on(event: string, listener: (...args: any[]) => void): this;
    off(event: string, listener: (...args: any[]) => void): this;
    removeListener(event: string, listener: (...args: any[]) => void): this;

    pause(): void;
    resume(): void;
}

export interface PoolConnection extends Connection {
    release(): void;
}

export interface Pool {
    getConnection(callback: (err: Error | null, connection?: PoolConnection) => void): void;
    releaseConnection(connection: PoolConnection): void;
    end(callback?: (err: Error | null) => void): void;

    query(sql: string, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    query(sql: string, values: any[], callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    query(options: QueryOptions, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;

    execute(sql: string, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    execute(sql: string, values: any[], callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;
    execute(options: QueryOptions, callback: (err: Error | null, results?: any[], fields?: FieldInfo[]) => void): void;

    on(event: string, listener: (...args: any[]) => void): this;
    off(event: string, listener: (...args: any[]) => void): this;
    removeListener(event: string, listener: (...args: any[]) => void): this;
}

// Promise-based interfaces
export interface PromiseConnection {
    connect(): Promise<void>;
    end(): Promise<void>;
    destroy(): void;

    query(sql: string): Promise<[any[], FieldInfo[]]>;
    query(sql: string, values: any[]): Promise<[any[], FieldInfo[]]>;
    query(options: QueryOptions): Promise<[any[], FieldInfo[]]>;

    execute(sql: string): Promise<[any[], FieldInfo[]]>;
    execute(sql: string, values: any[]): Promise<[any[], FieldInfo[]]>;
    execute(options: QueryOptions): Promise<[any[], FieldInfo[]]>;

    beginTransaction(): Promise<void>;
    commit(): Promise<void>;
    rollback(): Promise<void>;

    changeUser(options: any): Promise<void>;
    ping(): Promise<void>;
    statistics(): Promise<ConnectionStats>;

    format(sql: string, values?: any[]): string;
    escape(value: any): string;
    escapeId(identifier: string): string;
}

export interface PromisePoolConnection extends PromiseConnection {
    release(): void;
}

export interface PromisePool {
    getConnection(): Promise<PromisePoolConnection>;
    end(): Promise<void>;

    query(sql: string): Promise<[any[], FieldInfo[]]>;
    query(sql: string, values: any[]): Promise<[any[], FieldInfo[]]>;
    query(options: QueryOptions): Promise<[any[], FieldInfo[]]>;

    execute(sql: string): Promise<[any[], FieldInfo[]]>;
    execute(sql: string, values: any[]): Promise<[any[], FieldInfo[]]>;
    execute(options: QueryOptions): Promise<[any[], FieldInfo[]]>;
}

// Main module interface
export interface MySQL2Module {
    createConnection(options: ConnectionOptions): Connection;
    createPool(options: ConnectionOptions): Pool;
    createConnectionPromise(options: ConnectionOptions): PromiseConnection;
    createPoolPromise(options: ConnectionOptions): PromisePool;

    format(sql: string, values?: any[]): string;
    escape(value: any): string;
    escapeId(identifier: string): string;
    raw(value: any): any;

    // Promise-based API
    promise: {
        createConnection(options: ConnectionOptions): PromiseConnection;
        createPool(options: ConnectionOptions): PromisePool;
        format(sql: string, values?: any[]): string;
        escape(value: any): string;
        escapeId(identifier: string): string;
        raw(value: any): any;
    };
}

// Function declarations
export declare function createConnection(options: ConnectionOptions): Connection;
export declare function createPool(options: ConnectionOptions): Pool;
export declare function createConnectionPromise(options: ConnectionOptions): PromiseConnection;
export declare function createPoolPromise(options: ConnectionOptions): PromisePool;

export declare function format(sql: string, values?: any[]): string;
export declare function escape(value: any): string;
export declare function escapeId(identifier: string): string;
export declare function raw(value: any): any;

// Promise-based exports
export declare const promise: {
    createConnection(options: ConnectionOptions): PromiseConnection;
    createPool(options: ConnectionOptions): PromisePool;
    format(sql: string, values?: any[]): string;
    escape(value: any): string;
    escapeId(identifier: string): string;
    raw(value: any): any;
};

// Default export
declare const mysql2: MySQL2Module;
export default mysql2;
