/*
 * File Upload Example Route
 *
 * This route demonstrates how to use the file upload plugin
 * to handle binary file uploads in TurboScript using turboPlugin().
 *
 * The turboPlugin() function provides type-safe access to plugins
 * and is the recommended approach over require().
 */
export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get the file upload plugin using turboPlugin
        const fileUpload = turboPlugin<FileUploadPlugin>('fileupload');

        // Upload a test file using base64 data
        const result = await fileUpload.saveBase64("SGVsbG8gV29ybGQ=", "test.txt", {
            directory: "demo",
            generateHash: true,
            allowedTypes: ["text/plain", "application/octet-stream"],
            maxSize: 1024 * 1024 // 1MB
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "File uploaded successfully using turboPlugin",
                data: result
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "File upload failed",
                stack: error instanceof Error ? error.stack : undefined
            }
        };
    }
};
