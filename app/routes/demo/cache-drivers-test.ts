import { turboCache } from "../../utils/turbo-cache";

export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    const drivers = ["memory-local", "redis-server", "memcached-server", "file-system"];
    const results: Record<string, unknown> = {};

    await Promise.all(
        drivers.map(async (driverName) => {
            try {
                const cache = turboCache(driverName);
                const key = `test:${driverName}:${Date.now()}`;
                const testValue = { driver: driverName, timestamp: Date.now(), data: "test" };

                // Test set operation
                await cache.set(key, testValue, 60);

                // Test get operation
                const retrieved = await cache.get(key);

                // Test has operation
                const exists = await cache.has(key);

                // Test del operation
                await cache.del(key);
                const existsAfterDel = await cache.has(key);

                results[driverName] = {
                    status: "success",
                    set: "ok",
                    get: retrieved ? "ok" : "fail",
                    has: exists ? "ok" : "fail",
                    del: existsAfterDel ? "fail" : "ok",
                    retrieved: JSON.stringify(retrieved),
                    expected: JSON.stringify(testValue),
                };
            } catch (err) {
                results[driverName] = {
                    status: "error",
                    error: err instanceof Error ? err.message : err
                };
            }
        })
    );

    return {
        code: 200,
        response: {
            status: "driver-test-results",
            results
        }
    };
};
