import { turboCache } from "../../utils/turbo-cache";

export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    const cache = turboCache("memory-local");
    const key = "demo:test";
    const jsonKey = "demo:json";
    const results: Record<string, unknown> = {};

    // 1. Set string value
    try {
        await cache.set(key, { time: Date.now(), value: "bar" }, 60);
        results.setString = "ok";
    } catch (err) {
        results.setString = err instanceof Error ? err.message : err;
    }

    // 2. Set JSON value
    const obj = { foo: "bar", n: 42, arr: [1, 2, 3], nested: { a: 1 } };
    try {
        await cache.set(jsonKey, obj, 60);
        results.setJson = "ok";
    } catch (err) {
        results.setJson = err instanceof Error ? err.message : err;
    }

    // 3. Get string value
    try {
        const cached = await cache.get(key);
        results.getString = cached === "bar" ? "ok" : cached;
    } catch (err) {
        results.getString = err instanceof Error ? err.message : err;
    }

    // 4. Get JSON value
    try {
        const cachedJson = await cache.get(jsonKey);
        const sort = (v: unknown): string =>
            typeof v === "object" && v !== null
                ? JSON.stringify(v, Object.keys(v).sort())
                : JSON.stringify(v);
        results.getJson = sort(cachedJson) === sort(obj) ? "ok" : cachedJson;
    } catch (err) {
        results.getJson = err instanceof Error ? err.message : err;
    }

    // 5. Has operation
    try {
        results.hasString = await cache.has(key) ? "ok" : "miss";
        results.hasJson = await cache.has(jsonKey) ? "ok" : "miss";
    } catch (err) {
        results.hasString = err instanceof Error ? err.message : err;
        results.hasJson = err instanceof Error ? err.message : err;
    }

    // 6. Del operation
    try {
        await cache.del(key);
        results.delString = "ok";
        const keyExists = await cache.has(key);
        results.hasStringAfterDel = keyExists ? "fail" : "ok";
    } catch (err) {
        results.delString = err instanceof Error ? err.message : err;
    }

    // 7. Flush operation
    try {
        await cache.flush();
        results.flush = "ok";
        results.hasJsonAfterFlush = await cache.has(jsonKey) ? "fail" : "ok";
    } catch (err) {
        results.flush = err instanceof Error ? err.message : err;
    }

    return {
        code: 200,
        response: {
            status: "summary",
            results
        }
    };
};
