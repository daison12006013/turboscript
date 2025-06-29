# NOTICE

TurboScript
Copyright © 2025 TurboScript Project Contributors

This product includes software developed by the TurboScript Project Contributors.

## Third-Party Components

This software incorporates the following third-party components:

| Component                          | License      | Attribution Requirements                                | Used In                                                                          |
|------------------------------------|--------------|---------------------------------------------------------|----------------------------------------------------------------------------------|
| github.com/minio/minio             | AGPL-3.0     | Include AGPL-3.0 license text and copyright notice      | docker-compose.dev.yml (for Development purposes only!)                          |
| github.com/bradfitz/gomemcache     | Apache-2.0   | Preserve NOTICE file, copyright headers in source files | internal/tsengine/turbocache_utils.go                                            |
| github.com/dop251/goja             | MIT          | Include MIT license text and copyright notice           | internal/tsengine/*.go, main.go, main_test.go, README.md, SECURITY.md            |
| github.com/dop251/goja_nodejs      | MIT          | Include MIT license text and copyright notice           | internal/tsengine/*.go                                                           |
| github.com/evanw/esbuild           | MIT          | Include MIT license text and copyright notice           | internal/tsengine/compiler_utils.go, scripts/build-frontend.ts                   |
| github.com/fasthttp/router         | MIT          | Include MIT license text and copyright notice           | internal/server/routing.go                                                       |
| github.com/lib/pq                  | MIT          | Include MIT license text and copyright notice           | internal/config/loader.go, internal/config/database_manager.go                   |
| github.com/redis/go-redis/v9       | BSD-2-Clause | Include BSD-2-Clause license text and copyright notice  | internal/tsengine/turbocache_utils.go                                            |
| github.com/russross/blackfriday/v2 | BSD-2-Clause | Include BSD-2-Clause license text and copyright notice  | internal/templating/engine.go                                                    |
| github.com/valyala/fasthttp        | MIT          | Include MIT license text and copyright notice           | internal/server/server.go, internal/server/response.go                           |
| golang.org/x/crypto                | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/tsengine/crypto_utils.go                                                |
| gopkg.in/yaml.v3                   | Apache-2.0   | Preserve NOTICE file, copyright headers in source files | internal/config/loader.go                                                        |
| github.com/andybalholm/brotli      | MIT          | Include MIT license text and copyright notice           | internal/server/compression_test.go                                              |
| github.com/cespare/xxhash/v2       | MIT          | Include MIT license text and copyright notice           | internal/tsengine/turbocache_utils.go                                            |
| github.com/dgryski/go-rendezvous   | MIT          | Include MIT license text and copyright notice           | internal/tsengine/turbocache_utils.go                                            |
| github.com/dlclark/regexp2         | MIT          | Include MIT license text and copyright notice           | internal/config/loader.go                                                        |
| github.com/dop251/base64dec        | MIT          | Include MIT license text and copyright notice           | internal/tsengine/crypto_utils.go                                                |
| github.com/go-sourcemap/sourcemap  | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/tsengine/compiler_utils.go                                              |
| github.com/google/pprof            | Apache-2.0   | Preserve NOTICE file, copyright headers in source files | main.go                                                                          |
| github.com/klauspost/compress      | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/server/compression_test.go                                              |
| github.com/savsgio/gotils          | MIT          | Include MIT license text and copyright notice           | internal/server/server.go                                                        |
| github.com/valyala/bytebufferpool  | MIT          | Include MIT license text and copyright notice           | internal/server/server.go                                                        |
| golang.org/x/net                   | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/server/server.go                                                        |
| golang.org/x/sys                   | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/server/server.go                                                        |
| golang.org/x/text                  | BSD-3-Clause | Include BSD-3-Clause license text and copyright notice  | internal/server/server.go                                                        |
| @tailwindcss/typography            | MIT          | Include MIT license text and copyright notice           | tailwind.config.js                                                               |
| @types/jsonwebtoken                | MIT          | Include MIT license text and copyright notice           | package.json, app/routes/auth/                                                   |
| @types/react                       | MIT          | Include MIT license text and copyright notice           | package.json, app/frontend/                                                      |
| @types/react-dom                   | MIT          | Include MIT license text and copyright notice           | package.json, app/frontend/                                                      |
| autoprefixer                       | MIT          | Include MIT license text and copyright notice           | postcss.config.js                                                                |
| bcryptjs                           | MIT          | Include MIT license text and copyright notice           | package.json, app/utils/password.ts                                              |
| jsonwebtoken                       | MIT          | Include MIT license text and copyright notice           | package.json, app/utils/jwt.ts                                                   |
| postcss                            | MIT          | Include MIT license text and copyright notice           | postcss.config.js                                                                |
| react                              | MIT          | Include MIT license text and copyright notice           | package.json, app/frontend/                                                      |
| react-dom                          | MIT          | Include MIT license text and copyright notice           | package.json, app/frontend/                                                      |
| react-router-dom                   | MIT          | Include MIT license text and copyright notice           | package.json, app/frontend/                                                      |
| @types/bcryptjs                    | MIT          | Include MIT license text and copyright notice           | package.json, app/utils/password.ts                                              |
| @types/node                        | MIT          | Include MIT license text and copyright notice           | package.json, tsconfig.json                                                      |
| @typescript-eslint/eslint-plugin   | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| @typescript-eslint/parser          | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| esbuild                            | MIT          | Include MIT license text and copyright notice           | scripts/build-frontend.ts, scripts/build.ts, internal/tsengine/compiler_utils.go |
| eslint                             | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| eslint-plugin-import               | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| eslint-plugin-prefer-arrow         | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| eslint-plugin-unicorn              | MIT          | Include MIT license text and copyright notice           | package.json, eslint.config.js                                                   |
| newman                             | Apache-2.0   | Preserve NOTICE file, copyright headers in source files | postman/README.md                                                                |
| tailwindcss                        | MIT          | Include MIT license text and copyright notice           | tailwind.config.js, package.json                                                 |
| tsx                                | MIT          | Include MIT license text and copyright notice           | package.json, scripts/build.ts, scripts/build-frontend.ts                        |
| typescript                         | Apache-2.0   | Preserve NOTICE file, copyright headers in source files | package.json, tsconfig.json                                                      |
| typescript-lit-html-plugin         | MIT          | Include MIT license text and copyright notice           | tsconfig.json                                                                    |

*All third-party components are subject to their original license terms.*

## Required Attribution

When distributing this software or derivative works:

1. Preserve this NOTICE file
2. Maintain all copyright notices from source files
3. Retain original LICENSE files from third-party components
4. For Apache-2.0 components:
   - Preserve any included NOTICE files
   - Keep copyright headers in source files
5. Create `licenses/` directory containing all third-party license texts

---

Repository: <https://github.com/daison12006013/turboscript>
Contact: <daison12006013@gmail.com>
