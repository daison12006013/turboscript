export const handle = async (_event: Event): Promise<TurboScriptResponse> => ({
    code: 200,
    type: 'html',
    response: await turboHtml('demo-broadcasting/sse/index.html')
});
