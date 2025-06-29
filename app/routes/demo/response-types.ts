export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const { type } = event.queryParameters;

    try {
        switch (type) {
            case 'html':
                return {
                    code: 200,
                    response: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TurboScript HTML Response</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 30px;
            box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
        }
        h1 {
            color: #fff;
            text-align: center;
            margin-bottom: 30px;
        }
        .feature {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 10px;
            padding: 15px;
            margin: 15px 0;
            border-left: 4px solid #00ff88;
        }
        .highlight {
            background: rgba(0, 255, 136, 0.2);
            padding: 2px 6px;
            border-radius: 4px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 TurboScript HTML Response</h1>
        <div class="feature">
            <h3>✨ Clean HTML Output</h3>
            <p>This HTML response is automatically cleaned by the <span class="highlight">Go server layer</span> to remove unwanted whitespace, newlines, and tabs for optimal performance.</p>
        </div>
        <div class="feature">
            <h3>🎯 Content-Type Detection</h3>
            <p>The server automatically sets the <span class="highlight">content-type header</span> to <code>text/html</code> when it detects HTML responses.</p>
        </div>
        <div class="feature">
            <h3>⚡ TurboScript Framework</h3>
            <p>Hybrid TypeScript + Go framework with <span class="highlight">async database queries</span> and automatic response optimization.</p>
        </div>
    </div>
</body>
</html>`,
                    type: 'html'
                };

            case 'markdown':
                return {
                    code: 200,
                    response: `# TurboScript Markdown Response

## Features

### 🚀 **Automatic Content-Type Detection**
The server automatically sets \`content-type: text/markdown\` for markdown responses.

### ✨ **Clean Markdown Rendering**
- **Bold text** and *italic text* support
- Code blocks and \`inline code\`
- Lists and structured content
- Headers and formatting

### 🎯 **TypeScript + Go Hybrid**
\`\`\`typescript
// TypeScript defines the business logic
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const users = await turboQuery('SELECT * FROM users WHERE active = $1', [true]);
    return {
        code: 200,
        response: { users },
        type: 'json'
    };
};
\`\`\`

### ⚡ **Performance Benefits**
1. **Async Database Operations**: Non-blocking queries with \`turboQuery()\`
2. **Parallel Execution**: Multiple queries with \`Promise.all()\`
3. **Optimized Responses**: Automatic whitespace cleanup for HTML
4. **Fast Compilation**: TypeScript to JavaScript via esbuild

---

> **Note**: This markdown content is served directly by TurboScript with proper content-type headers.`,
                    type: 'markdown'
                };

            case 'markdown-html':
                return {
                    code: 200,
                    response: `# 🚀 TurboScript Markdown-to-HTML Response

This markdown content is **automatically converted to HTML** by the Go server layer using the blackfriday library!

## ✨ Key Features

### 🎯 **Automatic Conversion**
- Markdown content is parsed and converted to HTML
- Proper HTML structure with semantic elements
- Support for all common markdown syntax

### 📝 **Rich Content Support**
- **Bold** and *italic* text formatting
- \`inline code\` and code blocks
- Lists, headers, and links
- Tables, footnotes, and more

### 🔧 **TypeScript Integration**
\`\`\`typescript
// Just return markdown content with type: 'markdown-html'
return {
    code: 200,
    response: \`# Your markdown here\`,
    type: 'markdown-html'
};
\`\`\`

### ⚡ **Performance Benefits**
1. **Server-side Conversion**: No client-side JavaScript needed
2. **Clean HTML Output**: Whitespace automatically cleaned
3. **SEO Friendly**: Proper HTML structure for search engines
4. **Fast Rendering**: Pre-converted HTML loads instantly

---

> **Perfect for**: Documentation, blog posts, content management, and any text-rich applications!

**Table Example:**

| Feature | Status | Description |
|---------|--------|-------------|
| Markdown Parsing | ✅ | Full CommonMark support |
| HTML Conversion | ✅ | Clean, semantic HTML |
| Auto Linking | ✅ | URLs become clickable links |
| Code Highlighting | ✅ | Syntax highlighting ready |

**Footnote Support:** This is a reference[^1] to a footnote.

[^1]: This is the footnote content that appears at the bottom.`,
                    type: 'markdown-html'
                };

            case 'text':
                return {
                    code: 200,
                    response: `TurboScript Text Response
========================

This is a plain text response from TurboScript.

Key Features:
- Automatic content-type detection (text/plain)
- Clean text output without HTML tags
- Fast response times
- TypeScript + Go hybrid architecture

Performance Benefits:
✓ Async database operations with turboQuery()
✓ Parallel query execution with Promise.all()
✓ Optimized response handling
✓ Hot reloading during development

Server automatically sets content-type: text/plain for text responses.

Powered by TurboScript Framework 🚀`,
                    type: 'text'
                };

            default:
                return {
                    code: 200,
                    response: {
                        status: "success",
                        message: "TurboScript Response Type Demo",
                        available_types: [
                            "?type=html - Serve HTML content with auto-cleanup",
                            "?type=markdown - Serve raw markdown content",
                            "?type=markdown-html - Convert markdown to clean HTML",
                            "?type=text - Serve plain text content",
                            "?type=json - Serve JSON data (default)"
                        ],
                        examples: {
                            html: "/demo-response-types?type=html",
                            markdown: "/demo-response-types?type=markdown",
                            markdown_html: "/demo-response-types?type=markdown-html",
                            text: "/demo-response-types?type=text",
                            json: "/demo-response-types?type=json"
                        },
                        server_info: {
                            content_type: "application/json",
                            framework: "TurboScript",
                            language: "TypeScript + Go",
                            markdown_parser: "blackfriday/v2"
                        },
                        timestamp: new Date().toISOString()
                    },
                    type: 'json'
                };
        }
    } catch (error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: error instanceof Error ? error.message : "An unexpected error occurred"
            }
        };
    }
};
