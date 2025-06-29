import * as esbuild from 'esbuild';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { execSync } from 'node:child_process';

const isDev = process.env.NODE_ENV !== 'production';
const isWatch = process.argv.includes('--watch');
const isDocker = process.env.DOCKER_ENV === 'true' || fs.existsSync('/.dockerenv');
const frontendDir = './app/frontend';
const outDir = path.join(frontendDir, 'assets');

// Use esbuild-wasm in Docker environments to avoid platform issues
async function getEsbuild() {
    if (isDocker) {
        console.log('🐳 Docker environment detected, using esbuild-wasm...');
        try {
            const { initialize } = await import('esbuild-wasm');
            await initialize({
                wasmURL: 'https://unpkg.com/esbuild-wasm/esbuild.wasm'
            });
            return await import('esbuild-wasm');
        } catch (_error) {
            console.warn('⚠️  esbuild-wasm not available, falling back to regular esbuild...');
            return esbuild;
        }
    }
    return esbuild;
}

async function buildTailwindCSS() {
    console.log('🎨 Building Tailwind CSS...');
    try {
        const postcssCmd = `npx postcss ${frontendDir}/src/styles.css -o ${outDir}/styles.css${isDev ? '' : ' --minify'}`;
        execSync(postcssCmd, {
            stdio: 'inherit'
        });
    } catch (_error) {
        console.error('❌ Failed to build Tailwind CSS:', _error);
        // Continue with React build even if Tailwind fails
        console.log('⚠️  Continuing without Tailwind CSS...');
    }
}

async function buildReactApp() {
    console.log('🏗️  Building React frontend...');

    // Get appropriate esbuild version
    const esbuildApi = await getEsbuild();

    // Ensure output directory exists
    if (!fs.existsSync(outDir)) {
        fs.mkdirSync(outDir, { recursive: true });
    }

    // Build Tailwind CSS
    await buildTailwindCSS();

    // Build React app with esbuild
    console.log('⚛️  Building React app...');
    try {
        const buildOptions: esbuild.BuildOptions = {
            entryPoints: [path.join(frontendDir, 'src/App.tsx')],
            bundle: true,
            outfile: path.join(outDir, 'app.js'),
            format: 'iife',
            target: 'es2020',
            minify: !isDev,
            sourcemap: isDev,
            jsx: 'automatic',
            jsxImportSource: 'react',
            define: {
                'process.env.NODE_ENV': JSON.stringify(isDev ? 'development' : 'production'),
            },
            loader: {
                '.tsx': 'tsx',
                '.ts': 'ts',
            },
        };

        if (isWatch) {
            console.log('👀 Starting React watch mode...');
            const ctx = await esbuildApi.context(buildOptions);

            // Watch for changes
            await ctx.watch();

            // Keep the process running
            console.log('✅ React frontend watcher started!');
            console.log('🔄 Watching for changes in app/frontend/...');
            process.on('SIGINT', async () => {
                console.log('\n🧹 Stopping React watcher...');
                await ctx.dispose();
                process.exit(0);
            });

            // Keep the process alive
            await new Promise(() => { });
        } else {
            await esbuildApi.build(buildOptions);
            console.log('✅ React frontend built successfully!');
        }
    } catch (_error) {
        console.error('❌ Failed to build React app:', _error);
        process.exit(1);
    }
}

// Run the build
buildReactApp().catch((_error) => {
    console.error('❌ Build failed:', _error);
    process.exit(1);
});
