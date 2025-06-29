interface TurboCache {
    get(key: string, options?: { driver?: string }): Promise<unknown | undefined>;
    set(key: string, value: unknown, ttlSeconds?: number, options?: { driver?: string }): Promise<void>;
    del(key: string, options?: { driver?: string }): Promise<void>;
    has(key: string, options?: { driver?: string }): Promise<boolean>;
    flush(options?: { driver?: string }): Promise<void>;
}

// The actual implementation will be injected by the Go runtime (goja)
declare const turboCacheImpl: TurboCache;

export function turboCache(driver?: string): TurboCache {
    if (!driver) return turboCacheImpl;
    // Return a proxy that always uses the specified driver
    return {
        get: (key, opts) => turboCacheImpl.get(key, { ...opts, driver }),
        set: (key, value, ttl, opts) => turboCacheImpl.set(key, value, ttl, { ...opts, driver }),
        del: (key, opts) => turboCacheImpl.del(key, { ...opts, driver }),
        has: (key, opts) => turboCacheImpl.has(key, { ...opts, driver }),
        flush: (opts) => turboCacheImpl.flush({ ...opts, driver }),
    };
}
