# TurboScript Build System

TurboScript now includes a powerful TypeScript build system using esbuild for fast compilation with configurable optimization options.

## Overview

The build system compiles TypeScript files from the `app/` directory into optimized JavaScript for production deployment. It supports:

- Fast TypeScript compilation with esbuild
- Configurable minification and optimization options
- Tree shaking to remove unused code
- Source map generation (configurable)
- Support for external dependencies
- Directory structure preservation
- Import resolution and bundling

## Usage

### Development Build

```bash
npm run build                    # Development build (readable code)
```

### Production Build

```bash
npm run build:prod              # Production build (minified)
make build-dist                 # Complete distribution package
```

### Help

```bash
npm run build -- --help         # Show all options
```

## Build Configuration

### Development Mode

- **Minify**: Disabled (for debugging)
- **Source Maps**: Disabled (for performance)
- **Tree Shaking**: Enabled
- **Bundle**: Enabled
- **Target**: ES2020
- **Format**: CommonJS
- **Platform**: Node.js
- **Drop Console**: Disabled
- **Drop Debugger**: Disabled

### Production Mode

- **Minify**: Enabled (smaller files)
- **Source Maps**: Disabled (for security)
- **Tree Shaking**: Enabled (remove unused code)
- **Bundle**: Enabled
- **Target**: ES2020
- **Format**: CommonJS
- **Platform**: Node.js
- **Drop Console**: Enabled (remove console.log)
- **Drop Debugger**: Enabled (remove debugger statements)

## Environment Variables

You can override build settings using environment variables:

```bash
# Force minification in development
TS_BUILD_MINIFY=true npm run build

# Enable source maps
TS_BUILD_SOURCEMAP=true npm run build

# Disable tree shaking
TS_BUILD_TREESHAKE=false npm run build

# Remove console statements
TS_BUILD_DROP_CONSOLE=true npm run build

# Remove debugger statements
TS_BUILD_DROP_DEBUGGER=true npm run build

# Disable bundling (individual files)
TS_BUILD_BUNDLE=false npm run build
```

## Examples

### Development with Source Maps

```bash
TS_BUILD_SOURCEMAP=true npm run build
```

### Production without Minification

```bash
NODE_ENV=production TS_BUILD_MINIFY=false npm run build
```

### Custom Build Configuration

```bash
TS_BUILD_MINIFY=true \
TS_BUILD_SOURCEMAP=inline \
TS_BUILD_DROP_CONSOLE=true \
npm run build
```

## Build Output

The build system outputs compiled JavaScript files to `dist/app/` while preserving the original directory structure:

```text
dist/app/
├── routes/
│   ├── index.js              # Compiled from routes/index.ts
│   ├── auth/
│   │   ├── login.js         # Compiled from auth/login.ts
│   │   └── ...
│   └── ...
├── utils/
│   ├── auth.js              # Compiled from utils/auth.ts
│   └── ...
├── jobs/
│   └── ...
└── global.d.ts              # Copied TypeScript declarations
```

## Features

### Tree Shaking

Automatically removes unused code from your bundles, resulting in smaller file sizes.

### External Dependencies

The following dependencies are marked as external and won't be bundled:

- `bcryptjs`
- `crypto`
- `node:*` modules (fs, path, os, util, etc.)

### Import Resolution

Supports standard TypeScript/JavaScript import resolution:

- File extensions: `.ts`, `.js`, `.json`
- Package.json fields: `main`, `module`
- Node.js conditions

### Error Handling

The build system provides detailed error messages and will fail if any TypeScript compilation errors occur.

## Performance

### Build Speed

- **Development**: ~0.05 seconds for 24 files
- **Production**: ~0.04 seconds for 24 files (with minification)

### File Size Comparison

- **Development**: Readable, unminified code
- **Production**: Minified code with ~70% size reduction

### Parallel Compilation

Files are compiled in parallel for maximum performance.

## Integration with TurboScript

### Runtime Configuration

The build system integrates with TurboScript's runtime configuration:

- **Development**: Uses TypeScript files directly (`prefer_ts: true`)
- **Production**: Uses compiled JavaScript files (`prefer_js: true`)

### Distribution Build

The `make build-dist` command:

1. Installs Node.js dependencies
2. Compiles TypeScript to optimized JavaScript
3. Builds Go binaries
4. Creates production configuration
5. Packages everything for deployment

## Troubleshooting

### Common Issues

esbuild not found

```bash
npm install
```

**TypeScript compilation errors**
Check your TypeScript code and fix any type errors.

**Import resolution errors**
Ensure all imports are correctly specified and files exist.

### Debug Mode

Enable verbose output:

```bash
npm run build -- --production  # See production configuration
```

## Future Enhancements

The build system is designed to be extensible. Future enhancements may include:

- Custom esbuild plugins
- CSS/SCSS processing
- Asset optimization
- Code splitting
- Bundle analysis
- Custom banner/footer injection

## Contributing

When contributing to the build system:

1. Test both development and production builds
2. Ensure backward compatibility
3. Update documentation for new features
4. Add environment variable overrides for new options
