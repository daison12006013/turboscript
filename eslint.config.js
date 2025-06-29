import js from '@eslint/js';
import typescript from '@typescript-eslint/eslint-plugin';
import typescriptParser from '@typescript-eslint/parser';
import importPlugin from 'eslint-plugin-import';
import unicorn from 'eslint-plugin-unicorn';
import { extractGlobalsFromTypeScript } from './scripts/eslint-globals-extractor.js';

const autoGlobals = extractGlobalsFromTypeScript('.');

export default [
  // Ignore patterns (replaces .eslintignore)
  {
    ignores: [
      'dist/',
      'node_modules/',
      '*.js',
      '!scripts/*.js',
      '!eslint.config.js',
      'coverage/',
      '.nyc_output',
      '*.log',
      '*.tmp',
      '*.temp',
      '.DS_Store',
      'Thumbs.db',
      'app/frontend/assets/app.js',
    ],
  },

  // Base ESLint configuration
  js.configs.recommended,

  // TypeScript files configuration
  {
    files: ['**/*.ts'],
    languageOptions: {
      parser: typescriptParser,
      parserOptions: {
        ecmaVersion: 2020,
        sourceType: 'module',
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
      globals: {
        console: 'readonly',
        process: 'readonly',
        Buffer: 'readonly',
        __dirname: 'readonly',
        __filename: 'readonly',
        module: 'readonly',
        require: 'readonly',
        exports: 'readonly',
        global: 'readonly',

        ...autoGlobals,
      },
    },
    plugins: {
      '@typescript-eslint': typescript,
      import: importPlugin,
      unicorn: unicorn,
    },
    rules: {
      // ===== STRICT TYPE SAFETY =====
      // Absolutely no 'any' allowed
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-assignment': 'error',
      '@typescript-eslint/no-unsafe-call': 'error',
      '@typescript-eslint/no-unsafe-member-access': 'error',
      '@typescript-eslint/no-unsafe-return': 'error',

      // Strict type checking
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/prefer-optional-chain': 'error',
      '@typescript-eslint/no-unnecessary-condition': 'error',
      '@typescript-eslint/no-unnecessary-type-assertion': 'error',
      '@typescript-eslint/no-non-null-assertion': 'error',

      // Function and variable typing
      '@typescript-eslint/explicit-function-return-type': 'error',
      '@typescript-eslint/explicit-module-boundary-types': 'error',
      '@typescript-eslint/no-inferrable-types': 'off', // Allow explicit types for clarity

      // ===== CODE QUALITY =====
      // Prevent unused code
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
      'no-unused-vars': 'off', // Use TypeScript version

      // Consistent coding style
      '@typescript-eslint/consistent-type-definitions': ['error', 'interface'],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        { prefer: 'type-imports', disallowTypeAnnotations: false },
      ],
      '@typescript-eslint/array-type': ['error', { default: 'array-simple' }],

      // Naming conventions
      '@typescript-eslint/naming-convention': [
        'error',
        {
          selector: 'variableLike',
          format: ['camelCase', 'UPPER_CASE'],
          leadingUnderscore: 'allow',
        },
        {
          selector: 'typeLike',
          format: ['PascalCase'],
        },
        {
          selector: 'interface',
          format: ['PascalCase'],
          custom: {
            regex: '^I[A-Z]',
            match: false,
          },
        },
      ],

      // ===== IMPORT RULES =====
      'import/no-unresolved': 'off', // TypeScript handles this
      'import/no-cycle': 'off', // Disabled due to resolver issues
      'import/no-self-import': 'off', // Disabled due to resolver issues
      'import/no-useless-path-segments': 'off', // Disabled due to resolver issues
      'import/order': 'off', // Disabled due to resolver issues

      // ===== ARROW FUNCTION PREFERENCES =====
      'arrow-body-style': ['error', 'as-needed'],

      // ===== UNICORN RULES (MODERN JS/TS) =====
      'unicorn/no-array-for-each': 'error',
      'unicorn/prefer-type-error': 'error',
      'unicorn/throw-new-error': 'error',
      'unicorn/prefer-node-protocol': 'error',
      'unicorn/prefer-module': 'error',
      'unicorn/no-process-exit': 'off', // Allow process.exit in Node.js apps
      'unicorn/filename-case': [
        'error',
        {
          cases: {
            kebabCase: true,
            camelCase: true,
          },
        },
      ],

      // ===== GENERAL CODE QUALITY =====
      'no-debugger': 'error',
      'no-alert': 'error',
      'no-var': 'error',
      'prefer-const': 'error',
      'prefer-template': 'error',
      'no-implicit-coercion': 'error',
      // 'no-console': 'warn',
      // 'no-magic-numbers': [
      //   'warn',
      //   {
      //     ignore: [-1, 0, 1, 2],
      //     ignoreArrayIndexes: true,
      //     ignoreDefaultValues: true,
      //   },
      // ],

      // Error handling
      'no-throw-literal': 'error',

      // Performance
      'no-await-in-loop': 'error',
      'require-atomic-updates': 'error',

      // Security
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      'no-script-url': 'error',

      // ===== DISABLED UNICORN RULES =====
      // These can be too strict for some use cases
      'unicorn/no-null': 'off', // Allow null in TypeScript
      'unicorn/prefer-top-level-await': 'off', // Not always appropriate
      'unicorn/no-array-reduce': 'off', // Reduce is useful
      'unicorn/no-nested-ternary': 'off', // Sometimes useful for complex conditions
    },
    settings: {
      'import/resolver': {
        typescript: {
          alwaysTryTypes: true,
          project: './tsconfig.json',
        },
        alias: {
          map: [
            ['@app', './app'],
            ['@app/*', './app/*'],
          ],
          extensions: ['.ts', '.js', '.d.ts']
        }
      },
    },
  },

  // Build scripts configuration
  {
    files: ['scripts/*.js', 'scripts/*.ts'],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: 'module',
      globals: {
        console: 'readonly',
        process: 'readonly',
        Buffer: 'readonly',
        __dirname: 'readonly',
        __filename: 'readonly',
        global: 'readonly',
      },
    },
    rules: {
      // Relax rules for build scripts
      'no-console': 'off',
      'unicorn/prefer-module': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      '@typescript-eslint/explicit-function-return-type': 'off',
      'unicorn/prevent-abbreviations': 'off',
    },
  },

  // Test files configuration
  {
    files: ['**/*.test.ts', '**/*.spec.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unsafe-assignment': 'warn',
      '@typescript-eslint/no-unsafe-call': 'warn',
      '@typescript-eslint/no-unsafe-member-access': 'warn',
      '@typescript-eslint/no-unsafe-return': 'warn',
      'no-magic-numbers': 'off',
      '@typescript-eslint/explicit-function-return-type': 'off',
    },
  },

  // Configuration files
  {
    files: ['*.config.js', '*.config.ts', 'eslint.config.js'],
    rules: {
      'unicorn/prefer-module': 'off',
      '@typescript-eslint/no-var-requires': 'off',
    },
  },

  // Global type definitions
  {
    files: ['**/*.d.ts'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off', // Type definitions may need any
      '@typescript-eslint/no-empty-interface': 'off',
      'unicorn/filename-case': 'off',
    },
  },
];
