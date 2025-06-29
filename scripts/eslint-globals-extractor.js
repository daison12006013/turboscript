/**
 * ESLint Global Extractor
 * Automatically extracts global declarations from TypeScript definition files
 * This is an example of how you could automate global detection
 */

import fs from 'node:fs';
import path from 'node:path';

/**
 * Extract global variables from TypeScript declaration files
 * @param {string} projectRoot - Root directory of the project
 * @returns {Record<string, 'readonly'>} - ESLint globals object
 */
export function extractGlobalsFromTypeScript(projectRoot) {
    const globals = {};

    // Read global.d.ts
    const globalDtsPath = path.join(projectRoot, 'app', 'global.d.ts');
    if (fs.existsSync(globalDtsPath)) {
        const content = fs.readFileSync(globalDtsPath, 'utf8');

        // Extract const declarations (simple regex - could be more sophisticated)
        const constMatches = content.match(/^\s*const\s+(\w+):/gm);
        if (constMatches) {
            for (const match of constMatches) {
                const name = match.match(/const\s+(\w+):/)?.[1];
                if (name) {
                    globals[name] = 'readonly';
                }
            }
        }

        // Extract interface declarations that might be global types
        const interfaceMatches = content.match(/^\s*interface\s+(\w+)/gm);
        if (interfaceMatches) {
            for (const match of interfaceMatches) {
                const name = match.match(/interface\s+(\w+)/)?.[1];
                if (name) {
                    globals[name] = 'readonly';
                }
            }
        }

        // Extract function declarations
        const functionMatches = content.match(/^\s*function\s+(\w+)\s*\(/gm);
        if (functionMatches) {
            for (const match of functionMatches) {
                const name = match.match(/function\s+(\w+)\s*\(/)?.[1];
                if (name) {
                    globals[name] = 'readonly';
                }
            }
        }
    }

    // Read goja modules
    const turbo_modulesPath = path.join(projectRoot, 'turbo_modules');
    if (fs.existsSync(turbo_modulesPath)) {
        const modules = fs.readdirSync(turbo_modulesPath);
        for (const module of modules) {
            const indexDtsPath = path.join(turbo_modulesPath, module, 'index.d.ts');
            if (fs.existsSync(indexDtsPath)) {
                const content = fs.readFileSync(indexDtsPath, 'utf8');

                // Extract global const declarations
                const globalConstMatches = content.match(/^\s*const\s+(\w+):/gm);
                if (globalConstMatches) {
                    for (const match of globalConstMatches) {
                        const name = match.match(/const\s+(\w+):/)?.[1];
                        if (name) {
                            globals[name] = 'readonly';
                        }
                    }
                }
            }
        }
    }

    return globals;
}

// Example usage:
if (import.meta.url === `file://${process.argv[1]}`) {
    const projectRoot = process.cwd();
    const extractedGlobals = extractGlobalsFromTypeScript(projectRoot);
    console.log('Extracted globals:', extractedGlobals);
}
