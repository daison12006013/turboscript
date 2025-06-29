export const meta = (event: Event): Record<string, unknown> => ({
    meta: {
        timestamp: new Date().toISOString(),
        headers: event.headers,
        queryParameters: event.queryParameters,
        pathParameters: event.pathParameters,
        body: event.body,
    },
});