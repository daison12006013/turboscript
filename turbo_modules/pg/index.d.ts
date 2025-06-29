// TypeScript definitions for @turboscript/pg
// Compatible with Node.js pg library

export interface ConnectionConfig {
    host?: string;
    port?: number;
    database?: string;
    user?: string;
    password?: string;
    connectionString?: string;
    ssl?: boolean | string | object;
    sslmode?: 'disable' | 'allow' | 'prefer' | 'require' | 'verify-ca' | 'verify-full';
    connectionTimeoutMillis?: number;
    queryTimeoutMillis?: number;
    idleTimeoutMillis?: number;
    max?: number;
    idleCount?: number;
    statement_timeout?: number;
    lock_timeout?: number;
    idle_in_transaction_session_timeout?: number;
    application_name?: string;
}

export interface Field {
    name: string;
    tableID: number;
    columnID: number;
    dataTypeID: number;
    dataTypeSize: number;
    typeModifier: number;
    format: string;
}

export interface QueryResult<T = any> {
    rows: T[];
    rowCount: number;
    command: string;
    fields: Field[];
    processingTimeMs: number;
}

export interface QueryArrayResult<T = any[]> {
    rows: T[];
    rowCount: number;
    command: string;
    fields: Field[];
    processingTimeMs: number;
}

export interface Notification {
    processId: number;
    channel: string;
    payload?: string;
}

export interface QueryConfig {
    name?: string;
    text: string;
    values?: any[];
    rowMode?: 'array' | 'object';
    types?: any;
}

export interface NativeBindings {
    name: string;
    version: string;
}

export interface ClientBase {
    query<T = any>(text: string): Promise<QueryResult<T>>;
    query<T = any>(text: string, values: any[]): Promise<QueryResult<T>>;
    query<T = any>(config: QueryConfig): Promise<QueryResult<T>>;

    on(event: 'connect', listener: () => void): this;
    on(event: 'end', listener: () => void): this;
    on(event: 'error', listener: (err: Error) => void): this;
    on(event: 'notice', listener: (notice: any) => void): this;
    on(event: 'notification', listener: (message: Notification) => void): this;
    on(event: 'drain', listener: () => void): this;
    on(event: string, listener: (...args: any[]) => void): this;

    off(event: string, listener: (...args: any[]) => void): this;
    removeListener(event: string, listener: (...args: any[]) => void): this;

    escapeIdentifier(str: string): string;
    escapeLiteral(str: string): string;
}

export interface ClientInterface extends ClientBase {
    connect(): Promise<void>;
    end(): Promise<void>;
    release(): Promise<void>;

    copyFrom(queryText: string): any; // Placeholder
    copyTo(queryText: string): any; // Placeholder
    pauseDrain(): void;
    resumeDrain(): void;
}

export interface PoolClient extends ClientBase {
    release(err?: Error | boolean): void;
}

export interface PoolInterface extends ClientBase {
    connect(): Promise<PoolClient>;
    end(): Promise<void>;

    totalCount: number;
    idleCount: number;
    waitingCount: number;
}

export declare const defaults: Required<Pick<ConnectionConfig,
    'host' | 'port' | 'database' | 'user' | 'password' | 'ssl' | 'sslmode' |
    'connectionTimeoutMillis' | 'queryTimeoutMillis' | 'idleTimeoutMillis' |
    'max' | 'idleCount' | 'application_name'>>;

export declare const native: NativeBindings;

export declare class Client implements ClientInterface {
    constructor(config?: string | ConnectionConfig);
    connect(): Promise<void>;
    end(): Promise<void>;
    release(): Promise<void>;
    query<T = any>(text: string): Promise<QueryResult<T>>;
    query<T = any>(text: string, values: any[]): Promise<QueryResult<T>>;
    query<T = any>(config: QueryConfig): Promise<QueryResult<T>>;
    on(event: string, listener: (...args: any[]) => void): this;
    off(event: string, listener: (...args: any[]) => void): this;
    removeListener(event: string, listener: (...args: any[]) => void): this;
    copyFrom(queryText: string): any;
    copyTo(queryText: string): any;
    pauseDrain(): void;
    resumeDrain(): void;
    escapeIdentifier(str: string): string;
    escapeLiteral(str: string): string;
}

export declare class Pool implements PoolInterface {
    constructor(config?: string | ConnectionConfig);
    connect(): Promise<PoolClient>;
    end(): Promise<void>;
    query<T = any>(text: string): Promise<QueryResult<T>>;
    query<T = any>(text: string, values: any[]): Promise<QueryResult<T>>;
    query<T = any>(config: QueryConfig): Promise<QueryResult<T>>;
    on(event: string, listener: (...args: any[]) => void): this;
    off(event: string, listener: (...args: any[]) => void): this;
    removeListener(event: string, listener: (...args: any[]) => void): this;
    escapeIdentifier(str: string): string;
    escapeLiteral(str: string): string;
    readonly totalCount: number;
    readonly idleCount: number;
    readonly waitingCount: number;
}

// Default export is Client constructor for compatibility
declare const pg: {
    Client: typeof Client;
    Pool: typeof Pool;
    defaults: typeof defaults;
    native: NativeBindings;
    default: typeof Client;
};

export default pg;
