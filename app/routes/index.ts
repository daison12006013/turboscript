import { meta } from '../utils/meta';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const contentType = event.headers['content-type'] || event.headers['Content-Type'];

        if (contentType === 'application/json') {
            return {
                code: 200,
                response: {
                    status: "success",
                    message: "Welcome to TurboScript!",
                    meta: {
                        timestamp: new Date().toISOString(),
                        queryParameters: event.queryParameters,
                        pathParameters: event.pathParameters,
                    }
                }
            };
        }

        return {
            code: 200,
            type: 'html',
            response: await turboHtml('static/welcome.html', {
                title: 'Welcome to TurboScript!',
            }),
        };
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: "An unexpected error occurred",
                errors: [error instanceof Error ? error.message : "Unknown error"],
                ...meta(event),
            }
        };
    }
};