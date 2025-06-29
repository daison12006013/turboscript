# XSS Protection Implementation

## Overview

TurboScript uses `template.JSEscapeString` for XSS protection when embedding JSON data in HTML templates for React applications. This provides better security and proper JavaScript context escaping compared to HTML-specific escaping functions.

## Implementation Details

### Function: `SanitizeJSONForHTML`

The `SanitizeJSONForHTML` function in `internal/server/react.go` uses Go's `text/template.JSEscapeString` function to escape JSON data before embedding it in HTML templates.

```go
func SanitizeJSONForHTML(jsonData string) string {
    return template.JSEscapeString(jsonData)
}
```

### Why `template.JSEscapeString`?

- **JavaScript Context**: Since the JSON data is embedded in a `<script>` tag, it needs JavaScript-specific escaping
- **Unicode Escaping**: Converts dangerous HTML characters to safe Unicode sequences (e.g., `<` becomes `\u003C`)
- **Quote Escaping**: Properly escapes quotes for JavaScript string context
- **XSS Prevention**: Prevents execution of malicious scripts embedded in JSON data

### Escaping Examples

| Input | Output |
|-------|--------|
| `<script>` | `\u003Cscript\u003E` |
| `&` | `\u0026` |
| `'` | `\'` |
| `"` | `\"` |

## Testing

The XSS protection is thoroughly tested in `internal/server/react_test.go`:

### Test Cases

1. **Normal JSON data** - Ensures regular data passes through correctly
2. **Script tag injection** - Verifies `<script>` tags are properly escaped
3. **HTML entities** - Tests ampersands and angle brackets
4. **JavaScript event handlers** - Checks onclick and similar attributes
5. **Iframe injection** - Prevents iframe-based XSS
6. **Quotes and backslashes** - Ensures proper string escaping

### XSS Prevention Verification

The tests verify that:

- Literal `<script>` tags are not present in output
- Dangerous JavaScript is properly escaped
- Unicode escaping is applied to HTML characters
- The output is safe for JavaScript context

## Security Considerations

### What It Protects Against

- Script injection via JSON data
- HTML tag injection
- JavaScript event handler injection
- Iframe-based XSS attacks

### Usage in React Applications

When serving React applications, initial data is embedded in the HTML template:

```html
<script>
    window.__INITIAL_DATA__ = {{.Data}};
</script>
```

The `{{.Data}}` template variable contains JSON data sanitized using `SanitizeJSONForHTML` to prevent XSS attacks.

## Best Practices

1. **Always sanitize** JSON data before embedding in HTML templates
2. **Use JavaScript-specific escaping** for script contexts
3. **Test thoroughly** with various XSS payloads
4. **Validate input** at the data source when possible
5. **Use Content Security Policy** as an additional layer of protection

## Performance

The `template.JSEscapeString` function is highly optimized and suitable for production use. Benchmark tests show minimal performance impact for typical JSON payloads.
