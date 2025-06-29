/*
 * File Upload Test Route
 *
 * This route tests file upload with different file types and shows
 * the complete upload functionality working.
 */
export const handle = async (_event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get the file upload plugin using turboPlugin
        const fileUpload = turboPlugin<FileUploadPlugin>('fileupload');

        // Test multiple file uploads with different types
        const results = [];

        // Upload a JSON file (base64 encoded)
        const jsonData = "eyJtZXNzYWdlIjoiSGVsbG8gZnJvbSBKU09OIGZpbGUiLCJ0aW1lc3RhbXAiOiIyMDI1LTA3LTEzIn0=";
        const jsonResult = await fileUpload.saveBase64(jsonData, "test.json", {
            directory: "test-uploads",
            generateHash: true,
            allowedTypes: ["application/json", "text/plain"],
            maxSize: 1024 * 1024
        });
        results.push({ type: "JSON", result: jsonResult });

        // Upload an HTML file (base64 encoded)
        const htmlData = "PGh0bWw+PGJvZHk+PGgxPkhlbGxvIFdvcmxkPC9oMT48L2JvZHk+PC9odG1sPg==";
        const htmlResult = await fileUpload.saveBase64(htmlData, "test.html", {
            directory: "test-uploads",
            generateHash: true,
            allowedTypes: ["text/html", "text/plain"],
            maxSize: 1024 * 1024
        });
        results.push({ type: "HTML", result: htmlResult });

        // Upload a text file (base64 encoded)
        const textData = "VGhpcyBpcyBhIHRlc3QgdGV4dCBmaWxlLgpMaW5lIDIKTGluZSAz";
        const textResult = await fileUpload.saveBase64(textData, "test.txt", {
            directory: "test-uploads",
            generateHash: true,
            allowedTypes: ["text/plain", "application/octet-stream"],
            maxSize: 1024 * 1024
        });
        results.push({ type: "TEXT", result: textResult });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Multiple files uploaded successfully",
                data: {
                    uploads: results,
                    totalFiles: results.length
                }
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "File upload failed",
                errorDetails: error instanceof Error ? error.stack : undefined
            }
        };
    }
};
