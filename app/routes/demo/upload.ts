/*
 * Binary File Upload Route
 *
 * This route demonstrates how to handle actual binary file uploads
 * from multipart/form-data requests using TurboScript's file upload plugin.
 *
 * Supports:
 * - Real binary file uploads via multipart/form-data
 * - Multiple file upload methods (base64, binary data)
 * - Configurable upload options (directory, file types, size limits)
 * - File metadata extraction and validation
 */

interface UploadedFile {
    filename: string;
    data: string | Uint8Array | Buffer;
    size?: number;
    mimeType?: string;
}

interface MultipartRequestBody {
    files?: UploadedFile[];
    fields?: Record<string, string>;
}

interface UploadResult {
    originalName: string;
    size?: number;
    mimeType?: string;
    uploadResult?: FileInfo;
    error?: string;
}

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Get the file upload plugin using turboPlugin
        const fileUpload = turboPlugin<FileUploadPlugin>('fileupload');

        // Check if this is a binary file upload request
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        const contentType = event.headers['content-type'] ?? event.headers['Content-Type'] ?? '';

        if (contentType.includes('multipart/form-data')) {
            // Handle multipart/form-data binary uploads
            return await handleMultipartUpload(event, fileUpload);
        } else if (event.body.fileData && event.body.filename) {
            // Handle base64 data uploads via JSON body
            return await handleBase64Upload(event, fileUpload);
        } else if (contentType.startsWith('image/') || contentType.startsWith('video/') || contentType.startsWith('audio/') || contentType === 'application/pdf') {
            // Handle raw binary uploads (e.g., from --data-binary)
            return await handleRawBinaryUpload(event, fileUpload, contentType);
        } else {
            // Return demo/usage information
            return await showUploadDemo(fileUpload);
        }

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "File upload failed",
                stack: error instanceof Error ? error.stack : undefined,
                details: "Check Content-Type header and request body format"
            }
        };
    }
};

/**
 * Handle raw binary uploads (e.g., from --data-binary with specific Content-Type)
 */
async function handleRawBinaryUpload(event: Event, fileUpload: FileUploadPlugin, contentType: string): Promise<TurboScriptResponse> {
    // For raw binary uploads, the Go server provides the data in a specific format
    const binaryData = event.body.binaryData as string;
    const size = event.body.size as number;

    if (!binaryData) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "No binary data found in request body",
                contentType,
                bodyKeys: Object.keys(event.body),
                hint: "Make sure to send binary data with --data-binary flag"
            }
        };
    }

    // Get file extension from content type
    const getExtensionFromContentType = (mimeType: string): string => {
        const mimeToExt: Record<string, string> = {
            'image/jpeg': '.jpg',
            'image/png': '.png',
            'image/gif': '.gif',
            'image/webp': '.webp',
            'image/svg+xml': '.svg',
            'application/pdf': '.pdf',
            'video/mp4': '.mp4',
            'video/webm': '.webm',
            'audio/mp3': '.mp3',
            'audio/wav': '.wav'
        };
        return mimeToExt[mimeType] || '.bin';
    };

    // Generate filename with timestamp
    const timestamp = Date.now();
    const extension = getExtensionFromContentType(contentType);
    const filename = `upload-${timestamp}${extension}`;

    try {
        // Save the binary data (it's already base64 encoded from Go)
        const result = await fileUpload.saveBase64(binaryData, filename, {
            directory: "binary-uploads",
            generateHash: true,
            maxSize: 10 * 1024 * 1024, // 10MB
            allowedTypes: [
                "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
                "application/pdf", "text/plain", "text/csv",
                "application/json", "application/xml",
                "video/mp4", "video/webm", "audio/mp3", "audio/wav",
                "application/octet-stream"
            ]
        });

        return {
            code: 200,
            response: {
                status: "success",
                message: "Binary file uploaded successfully",
                uploadType: "raw-binary",
                contentType,
                originalSize: size,
                data: result
            }
        };

    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "Binary upload failed",
                contentType,
                filename,
                originalSize: size
            }
        };
    }
}

/**
 * Handle multipart/form-data binary file uploads
 */
async function handleMultipartUpload(event: Event, fileUpload: FileUploadPlugin): Promise<TurboScriptResponse> {
    // For now, extract file data from the parsed body
    // The Go server should parse multipart data and provide it in event.body
    const requestBody = event.body as MultipartRequestBody;
    const files = requestBody.files ?? [];
    const fields = requestBody.fields ?? {};

    if (files.length === 0) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "No files found in multipart request",
                hint: "Make sure to include files in your multipart/form-data request",
                expectedFormat: {
                    body: {
                        files: [
                            {
                                filename: "example.jpg",
                                data: "binary_data_or_base64_string",
                                size: 12345,
                                mimeType: "image/jpeg"
                            }
                        ],
                        fields: {
                            directory: "uploads",
                            generateHash: "true"
                        }
                    }
                }
            }
        };
    }

    // Process files concurrently using Promise.all to avoid await inside a loop
    const uploadResults: UploadResult[] = await Promise.all(
        files.map(async (file) => {
            try {
                // Extract upload options from form fields
                // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                const directory = fields.directory ?? "uploads";
                const generateHash = fields.generateHash !== "false";
                // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                const maxSize = parseInt(fields.maxSize ?? "10485760", 10); // 10MB default

                // Handle binary file data
                let result: FileInfo;

                if (file.data && file.filename) {
                    // If file data is provided as binary or base64
                    if (typeof file.data === 'string') {
                        // Assume base64 if string
                        result = await fileUpload.saveBase64(file.data, file.filename, {
                            directory,
                            generateHash,
                            maxSize,
                            allowedTypes: [
                                "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
                                "application/pdf", "text/plain", "text/csv",
                                "application/json", "application/xml",
                                "video/mp4", "video/webm", "audio/mp3", "audio/wav",
                                "application/octet-stream" // Allow generic binary files
                            ]
                        });
                    } else {
                        // Handle binary data (Uint8Array/Buffer)
                        result = await fileUpload.saveFile(file.data, {
                            filename: file.filename,
                            directory,
                            generateHash,
                            maxSize,
                            allowedTypes: [
                                "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
                                "application/pdf", "text/plain", "text/csv",
                                "application/json", "application/xml",
                                "video/mp4", "video/webm", "audio/mp3", "audio/wav",
                                "application/octet-stream" // Allow generic binary files
                            ]
                        });
                    }

                    return {
                        originalName: file.filename,
                        size: file.size ?? result.size,
                        mimeType: file.mimeType ?? result.mimeType,
                        uploadResult: result
                    };
                } else {
                    return {
                        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                        originalName: file.filename ?? "unknown",
                        error: "Missing file data or filename"
                    };
                }
            } catch (fileError) {
                return {
                    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                    originalName: file.filename ?? "unknown",
                    error: fileError instanceof Error ? fileError.message : "Upload failed"
                };
            }
        })
    );

    return {
        code: 200,
        response: {
            status: "success",
            message: `Processed ${files.length} file(s)`,
            uploadType: "multipart/form-data",
            data: {
                files: uploadResults,
                fields,
                summary: {
                    total: files.length,
                    successful: uploadResults.filter(r => !r.error).length,
                    failed: uploadResults.filter(r => r.error).length
                }
            }
        }
    };
}

/**
 * Handle base64 encoded file uploads via JSON body
 */
async function handleBase64Upload(event: Event, fileUpload: FileUploadPlugin): Promise<TurboScriptResponse> {
    const { fileData, filename, directory, generateHash, maxSize } = event.body;

    if (!fileData || !filename) {
        return {
            code: 400,
            response: {
                status: "error",
                message: "Missing required fields: fileData and filename",
                example: {
                    fileData: "SGVsbG8gV29ybGQ=",
                    filename: "example.txt",
                    directory: "uploads",
                    generateHash: true,
                    maxSize: 1048576
                }
            }
        };
    }

    const result = await fileUpload.saveBase64(fileData as string, filename as string, {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        directory: (directory as string) ?? "demo",
        generateHash: generateHash !== false,
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        maxSize: (maxSize as number) ?? 10 * 1024 * 1024, // 10MB
        allowedTypes: [
            "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
            "application/pdf", "text/plain", "text/csv",
            "application/json", "application/xml",
            "video/mp4", "video/webm", "audio/mp3", "audio/wav",
            "application/octet-stream" // Allow generic binary files
        ]
    });

    return {
        code: 200,
        response: {
            status: "success",
            message: "File uploaded successfully via base64",
            uploadType: "base64",
            data: result
        }
    };
}

/**
 * Show upload demo and usage information
 */
async function showUploadDemo(fileUpload: FileUploadPlugin): Promise<TurboScriptResponse> {
    // Create a demo file to show the plugin is working
    const demoResult = await fileUpload.saveBase64("SGVsbG8gZnJvbSBUdXJib1NjcmlwdCEgVGhpcyBpcyBhIGRlbW8gZmlsZS4=", "demo.txt", {
        directory: "demo",
        generateHash: true,
        allowedTypes: ["text/plain", "application/octet-stream"],
        maxSize: 1024 * 1024 // 1MB
    });

    return {
        code: 200,
        response: {
            status: "demo",
            message: "TurboScript File Upload Endpoint",
            demoFile: demoResult,
            usage: {
                multipartUpload: {
                    method: "POST",
                    contentType: "multipart/form-data",
                    description: "Upload binary files using multipart/form-data",
                    example: "Use Postman or curl with file attachment",
                    note: "Requires server-side multipart parsing (currently being implemented)"
                },
                base64Upload: {
                    method: "POST",
                    contentType: "application/json",
                    body: {
                        fileData: "SGVsbG8gV29ybGQ=",
                        filename: "example.txt",
                        directory: "uploads",
                        generateHash: true,
                        maxSize: 1048576
                    }
                }
            },
            supportedTypes: [
                "Images: JPEG, PNG, GIF, WebP, SVG",
                "Documents: PDF, TXT, CSV, JSON, XML",
                "Media: MP4, WebM, MP3, WAV"
            ],
            maxSize: "10MB (configurable per request)",
            storageLocation: "./uploads/ (configurable)"
        }
    };
}
