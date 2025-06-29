export const handle = async (_event: Event): Promise<TurboScriptResponse> => ({
    code: 200,
    response: {
        status: 'OK',
        message: 'This is the demo index route',
    },
})