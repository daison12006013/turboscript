import { meta } from "../utils/meta";

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const contentType = event.headers['content-type'] || event.headers['Content-Type'];

        if (contentType === 'application/json') {
            return {
                code: 404,
                response: {
                    status: "failure",
                    message: "Endpoint not found",
                }
            };
        }

        return {
            code: 404,
            type: 'html',
            response: await turboHtml('static/404.html', {
                title: '404 Not Found',
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
}