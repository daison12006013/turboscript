/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Copyright (c) 2025 TurboScript Project Contributors
 * Author: Daison Cariño <daison12006013@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on TurboScript: https://github.com/daison12006013/turboscript
 */

/**
 * TurboScript TypeScript Build System
 *
 * This build script compiles TypeScript files from the app/ directory into optimized JavaScript
 * for production deployment. It uses esbuild for fast compilation and supports configurable
 * options for minification, tree shaking, and bundling.
 *
 * Features:
 * - Fast TypeScript compilation with esbuild
 * - Configurable minification with granular controls
 * - Separate controls for whitespace, identifiers, and syntax minification
 * - Tree shaking to remove unused code
 * - Source map generation (configurable)
 * - Support for external dependencies
 * - Preserve directory structure
 * - Import resolution and bundling
 *
 * Usage:
 *   npm run build              # Development build
 *   npm run build:prod         # Production build (minified)
 *   npm run build -- --help    # Show help
 *
 * Environment Variables:
 *   NODE_ENV=production        # Enable production optimizations
 *   TS_BUILD_MINIFY=true      # Force overall minification
 *   TS_BUILD_MINIFY_WHITESPACE=true  # Remove whitespace only (safest)
 *   TS_BUILD_MINIFY_IDENTIFIERS=true # Rename variables (can cause issues)
 *   TS_BUILD_MINIFY_SYNTAX=true      # Compact syntax (usually safe)
 *   TS_BUILD_SOURCEMAP=true   # Enable source maps
 *   TS_BUILD_TREESHAKE=false  # Disable tree shaking
 */

import * as esbuild from 'esbuild';
import * as fs from 'node:fs';
import * as path from 'node:path';

// Project paths - use current working directory approach
const ROOT_DIR = process.cwd();
const APP_DIR = path.join(ROOT_DIR, 'app');
const DIST_DIR = path.join(ROOT_DIR, 'dist', 'app');
const TSCONFIG_PATH = path.join(ROOT_DIR, 'tsconfig.json');

// Build configuration interface
interface BuildConfig {
    // Core options
    minify: boolean;
    sourcemap: boolean | 'inline' | 'external' | 'both';
    treeShaking: boolean;
    bundle: boolean;
    target: string;
    format: 'cjs' | 'esm' | 'iife';
    platform: 'node' | 'browser' | 'neutral';

    // Granular minification options
    minifyWhitespace: boolean;
    minifyIdentifiers: boolean;
    minifySyntax: boolean;

    // Advanced options
    external: string[];
    keepNames: boolean;
    mangleProps?: string | RegExp;
    reserveProps?: string | RegExp; // Pattern for properties to exclude from mangling
    dropConsole: boolean;
    dropDebugger: boolean;
    legalComments: 'none' | 'inline' | 'eof' | 'linked' | 'external';
    charset: 'ascii' | 'utf8';
    pure: string[];
    ignoreAnnotations: boolean;
    preserveSymlinks: boolean;

    // Resolution options
    resolveExtensions: string[];
    mainFields: string[];
    conditions: string[];

    // Output options
    globalName?: string;
    publicPath?: string;
    banner?: string;
    footer?: string;

    // TypeScript options
    tsconfigRaw?: string;

    // JSX options (for future use)
    jsx: 'transform' | 'preserve' | 'automatic';
    jsxFactory?: string;
    jsxFragment?: string;
    jsxImportSource?: string;
}

// Default development configuration
const getDefaultConfig = (): BuildConfig => ({
    minify: false,
    minifyWhitespace: false,
    minifyIdentifiers: false,
    minifySyntax: false,
    sourcemap: false,
    treeShaking: false,
    bundle: true,
    target: 'es2020',
    format: 'cjs',
    platform: 'node',
    external: [
        'bcryptjs',
        'crypto',
        'node:crypto',
        'node:url',
        'node:fs',
        'node:path',
        'node:os',
        'node:util',
        'node:stream',
        'node:buffer',
        'node:events',
        'node:http',
        'node:https',
        'node:querystring',
        'node:zlib'
    ],
    keepNames: true,
    mangleProps: undefined,
    reserveProps: undefined, // Simplify - don't use property mangling
    dropConsole: false,
    dropDebugger: false,
    legalComments: 'none',
    charset: 'utf8',
    pure: [
        // Mark turbo functions as pure but don't mangle them
        'turboQuery',
        'turboEmail',
        'turboJob',
        'turboMarkdownHtml',
        'turboCache'
    ],
    ignoreAnnotations: false,
    preserveSymlinks: false,
    resolveExtensions: ['.ts', '.js', '.json'],
    mainFields: ['main', 'module'],
    conditions: ['node'],
    jsx: 'transform'
});

// Production configuration (optimized but safe for TurboScript)
const getProductionConfig = (): BuildConfig => {
    const config = getDefaultConfig();
    return {
        ...config,
        minify: false, // Disable overall minify to prevent function renaming
        minifyWhitespace: true, // Only remove whitespace
        minifyIdentifiers: false, // Never rename identifiers in production
        minifySyntax: true, // Compact syntax is usually safe
        sourcemap: true,
        dropConsole: true,
        dropDebugger: true,
        legalComments: 'none',
        treeShaking: true
    };
};

// Apply environment variable overrides
const applyEnvironmentOverrides = (config: BuildConfig): BuildConfig => {
    const envConfig = { ...config };

    // Check for environment variable overrides
    if (process.env.TS_BUILD_MINIFY !== undefined) {
        envConfig.minify = process.env.TS_BUILD_MINIFY === 'true';
    }

    // Granular minification controls
    if (process.env.TS_BUILD_MINIFY_WHITESPACE !== undefined) {
        envConfig.minifyWhitespace = process.env.TS_BUILD_MINIFY_WHITESPACE === 'true';
    }

    if (process.env.TS_BUILD_MINIFY_IDENTIFIERS !== undefined) {
        envConfig.minifyIdentifiers = process.env.TS_BUILD_MINIFY_IDENTIFIERS === 'true';
    }

    if (process.env.TS_BUILD_MINIFY_SYNTAX !== undefined) {
        envConfig.minifySyntax = process.env.TS_BUILD_MINIFY_SYNTAX === 'true';
    }

    if (process.env.TS_BUILD_SOURCEMAP !== undefined) {
        const value = process.env.TS_BUILD_SOURCEMAP;
        if (value === 'true') {
            envConfig.sourcemap = true;
        } else if (value === 'false') {
            envConfig.sourcemap = false;
        } else if (['inline', 'external', 'both'].includes(value)) {
            envConfig.sourcemap = value as 'inline' | 'external' | 'both';
        }
    }

    if (process.env.TS_BUILD_TREESHAKE !== undefined) {
        envConfig.treeShaking = process.env.TS_BUILD_TREESHAKE === 'true';
    }

    if (process.env.TS_BUILD_DROP_CONSOLE !== undefined) {
        envConfig.dropConsole = process.env.TS_BUILD_DROP_CONSOLE === 'true';
    }

    if (process.env.TS_BUILD_DROP_DEBUGGER !== undefined) {
        envConfig.dropDebugger = process.env.TS_BUILD_DROP_DEBUGGER === 'true';
    }

    if (process.env.TS_BUILD_BUNDLE !== undefined) {
        envConfig.bundle = process.env.TS_BUILD_BUNDLE === 'true';
    }

    if (process.env.TS_BUILD_RESERVE_PROPS !== undefined) {
        envConfig.reserveProps = process.env.TS_BUILD_RESERVE_PROPS;
    }

    return envConfig;
};

// Get all TypeScript files recursively
const getTypeScriptFiles = (dir: string, baseDir: string = dir): string[] => {
    const files: string[] = [];

    try {
        const entries = fs.readdirSync(dir, { withFileTypes: true });

        for (const entry of entries) {
            const fullPath = path.join(dir, entry.name);

            if (entry.isDirectory()) {
                // Skip node_modules and other unwanted directories
                if (entry.name !== 'node_modules' && !entry.name.startsWith('.')) {
                    files.push(...getTypeScriptFiles(fullPath, baseDir));
                }
            } else if (entry.isFile() && entry.name.endsWith('.ts') && !entry.name.endsWith('.d.ts')) {
                files.push(fullPath);
            }
        }
    } catch (error) {
        console.warn(`Warning: Could not read directory ${dir}:`, error);
    }

    return files;
};

// Create directory recursively if it doesn't exist
const ensureDirectoryExists = (filePath: string): void => {
    const dir = path.dirname(filePath);
    if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
    }
};

// Build a single TypeScript file
const buildFile = async (inputFile: string, config: BuildConfig): Promise<void> => {
    // Calculate relative path from app directory
    const relativePath = path.relative(APP_DIR, inputFile);
    const outputFile = path.join(DIST_DIR, relativePath.replace(/\.ts$/, '.js'));

    // Ensure output directory exists
    ensureDirectoryExists(outputFile);

    try {
        const buildOptions: esbuild.BuildOptions = {
            entryPoints: [inputFile],
            outfile: outputFile,
            bundle: config.bundle,
            minify: config.minify,
            minifyWhitespace: config.minifyWhitespace,
            minifyIdentifiers: config.minifyIdentifiers,
            minifySyntax: config.minifySyntax,
            sourcemap: config.sourcemap,
            target: config.target,
            format: config.format,
            platform: config.platform,
            external: config.external,
            keepNames: config.keepNames,
            legalComments: config.legalComments,
            charset: config.charset,
            pure: config.pure,
            ignoreAnnotations: config.ignoreAnnotations,
            preserveSymlinks: config.preserveSymlinks,
            resolveExtensions: config.resolveExtensions,
            mainFields: config.mainFields,
            conditions: config.conditions,
            treeShaking: config.treeShaking,
            jsx: config.jsx,
            write: true,
            metafile: false,
            logLevel: 'warning',
            absWorkingDir: ROOT_DIR
        };

        // Add drop options for console and debugger
        if (config.dropConsole || config.dropDebugger) {
            const dropOptions: esbuild.Drop[] = [];
            if (config.dropConsole) dropOptions.push('console');
            if (config.dropDebugger) dropOptions.push('debugger');
            (buildOptions as any).drop = dropOptions;
        }

        // Add optional configurations with proper type assertions
        if (config.mangleProps && typeof config.mangleProps === 'string') {
            (buildOptions as any).mangleProps = new RegExp(config.mangleProps);
        }

        if (config.reserveProps && typeof config.reserveProps === 'string') {
            (buildOptions as any).reserveProps = new RegExp(config.reserveProps);
        }

        // Ensure handle function is kept - critical for TurboScript runtime
        if (config.keepNames) {
            (buildOptions as any).keepNames = true;
            // Don't set mangleProps to avoid function renaming issues
        }

        if (config.globalName) {
            (buildOptions as any).globalName = config.globalName;
        }

        if (config.publicPath) {
            (buildOptions as any).publicPath = config.publicPath;
        }

        if (config.banner) {
            (buildOptions as any).banner = { js: config.banner };
        }

        if (config.footer) {
            (buildOptions as any).footer = { js: config.footer };
        }

        if (config.jsxFactory) {
            (buildOptions as any).jsxFactory = config.jsxFactory;
        }

        if (config.jsxFragment) {
            (buildOptions as any).jsxFragment = config.jsxFragment;
        }

        if (config.jsxImportSource) {
            (buildOptions as any).jsxImportSource = config.jsxImportSource;
        }

        // Set tsconfig
        if (fs.existsSync(TSCONFIG_PATH)) {
            (buildOptions as any).tsconfig = TSCONFIG_PATH;
        }

        if (config.tsconfigRaw) {
            (buildOptions as any).tsconfigRaw = config.tsconfigRaw;
        }

        await esbuild.build(buildOptions);

    } catch (error) {
        console.error(`❌ Failed to build ${relativePath}:`, error);
        throw error;
    }
};

// Copy non-TypeScript files (like .d.ts files and other assets)
const copyNonTSFiles = async (): Promise<void> => {
    const copyFile = (src: string, dest: string): void => {
        ensureDirectoryExists(dest);
        fs.copyFileSync(src, dest);
    };

    const copyDirectory = (srcDir: string, destDir: string): void => {
        if (!fs.existsSync(srcDir)) return;

        const entries = fs.readdirSync(srcDir, { withFileTypes: true });

        for (const entry of entries) {
            const srcPath = path.join(srcDir, entry.name);
            const destPath = path.join(destDir, entry.name);

            if (entry.isDirectory()) {
                copyDirectory(srcPath, destPath);
            } else if (entry.isFile() && !entry.name.endsWith('.ts')) {
                // Copy non-TypeScript files (including .d.ts files)
                copyFile(srcPath, destPath);
            } else if (entry.name.endsWith('.d.ts')) {
                // Explicitly copy TypeScript declaration files
                copyFile(srcPath, destPath);
            }
        }
    };

    copyDirectory(APP_DIR, DIST_DIR);
};

// Main build function
const build = async (): Promise<void> => {
    console.log('🚀 TurboScript TypeScript Build System\n');

    // Determine build mode
    const isProduction = process.env.NODE_ENV === 'production' || process.argv.includes('--production');
    const showHelp = process.argv.includes('--help') || process.argv.includes('-h');

    if (showHelp) {
        console.log(`
Usage: npm run build [options]

Options:
  --production                  Use production optimizations
  --help, -h                    Show this help message

Environment Variables:
  NODE_ENV=production           Enable production mode
  TS_BUILD_MINIFY=true          Force overall minification (enables all minify options)
  TS_BUILD_MINIFY_WHITESPACE=true    Remove whitespace (safest option)
  TS_BUILD_MINIFY_IDENTIFIERS=true   Rename variables to shorter names (can cause issues)
  TS_BUILD_MINIFY_SYNTAX=true        Make syntax more compact (usually safe)
  TS_BUILD_SOURCEMAP=true       Enable source maps
  TS_BUILD_TREESHAKE=false      Disable tree shaking  TS_BUILD_DROP_CONSOLE=true    Remove console statements
  TS_BUILD_DROP_DEBUGGER=true   Remove debugger statements
  TS_BUILD_BUNDLE=false         Disable bundling
  TS_BUILD_RESERVE_PROPS=pattern Reserve properties matching pattern from mangling

Minification Notes:
  • minifyWhitespace: Removes whitespace - safest option, good for reducing file size
  • minifyIdentifiers: Renames variables - can break code that relies on variable names
  • minifySyntax: Compacts syntax - usually safe, makes code more compact

  TurboScript functions (turboQuery, turboEmail, etc.) are automatically protected
  from minification to prevent runtime issues.

  Use granular options if regular minification causes issues with your code.

Examples:
  npm run build                      # Development build
  npm run build --production         # Production build
  NODE_ENV=production npm run build  # Production build via env var
  TS_BUILD_MINIFY=true npm run build # Force full minification
  TS_BUILD_MINIFY_WHITESPACE=true npm run build # Only remove whitespace
`);
        return;
    }

    console.log(`📋 Build Mode: ${isProduction ? 'Production' : 'Development'}`);

    // Get build configuration
    let config = isProduction ? getProductionConfig() : getDefaultConfig();
    config = applyEnvironmentOverrides(config);

    console.log('⚙️  Build Configuration:');
    console.log(`   • Minify: ${config.minify}`);
    console.log(`   • Minify Whitespace: ${config.minifyWhitespace}`);
    console.log(`   • Minify Identifiers: ${config.minifyIdentifiers}`);
    console.log(`   • Minify Syntax: ${config.minifySyntax}`);
    console.log(`   • Source Maps: ${config.sourcemap}`);
    console.log(`   • Tree Shaking: ${config.treeShaking}`);
    console.log(`   • Bundle: ${config.bundle}`);
    console.log(`   • Target: ${config.target}`);
    console.log(`   • Format: ${config.format}`);
    console.log(`   • Platform: ${config.platform}`);
    console.log(`   • Drop Console: ${config.dropConsole}`);
    console.log(`   • Drop Debugger: ${config.dropDebugger}`);
    console.log(`   • External: ${config.external.length} packages`);
    console.log('');

    // Clean and create dist directory
    if (fs.existsSync(DIST_DIR)) {
        fs.rmSync(DIST_DIR, { recursive: true, force: true });
    }
    fs.mkdirSync(DIST_DIR, { recursive: true });

    // Get all TypeScript files
    console.log('🔍 Scanning for TypeScript files...');
    const tsFiles = getTypeScriptFiles(APP_DIR);
    console.log(`📁 Found ${tsFiles.length} TypeScript files to compile\n`);

    if (tsFiles.length === 0) {
        console.log('⚠️  No TypeScript files found in app/ directory');
        return;
    }

    // Build all files
    console.log('🔨 Compiling TypeScript files...');
    const startTime = Date.now();
    let compiledCount = 0;
    let errorCount = 0;

    // Build files in parallel for better performance
    const buildPromises = tsFiles.map(async (file) => {
        try {
            await buildFile(file, config);
            compiledCount++;
            const relativePath = path.relative(APP_DIR, file);
            console.log(`✅ ${relativePath} → ${relativePath.replace(/\.ts$/, '.js')}`);
        } catch (_error) {
            errorCount++;
            console.error(`❌ Failed to compile ${path.relative(APP_DIR, file)}`);
        }
    });

    await Promise.all(buildPromises);

    // Copy non-TypeScript files
    console.log('\n📋 Copying non-TypeScript files...');
    await copyNonTSFiles();

    const endTime = Date.now();
    const duration = (endTime - startTime) / 1000;

    console.log('\n📊 Build Summary:');
    console.log(`   • Compiled: ${compiledCount} files`);
    console.log(`   • Errors: ${errorCount} files`);
    console.log(`   • Duration: ${duration.toFixed(2)}s`);
    console.log(`   • Output: ${path.relative(ROOT_DIR, DIST_DIR)}/`);

    if (errorCount > 0) {
        console.log('\n❌ Build completed with errors');
        process.exit(1);
    } else {
        console.log('\n✅ Build completed successfully!');
    }
};

// Handle uncaught errors
process.on('uncaughtException', (error) => {
    console.error('❌ Uncaught Exception:', error);
    process.exit(1);
});

process.on('unhandledRejection', (reason, _promise) => {
    console.error('❌ Unhandled Rejection:', reason);
    process.exit(1);
});

// Run the build
build().catch((error) => {
    console.error('❌ Build failed:', error);
    process.exit(1);
});

export { build, getDefaultConfig, getProductionConfig, type BuildConfig };
