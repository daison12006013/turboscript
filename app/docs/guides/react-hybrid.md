# React Hybrid Rendering (HYBRID)

TurboScript provides built-in support for React Hybrid Rendering (HYBRID) with automatic data injection and static asset serving. This feature allows you to build modern single-page applications with server-side rendered initial content for improved performance and SEO.

## Overview

The React HYBRID system in TurboScript includes:

- **Hybrid rendering** of React components with initial data
- **Automatic data loading** from configurable TypeScript endpoints
- **Static asset serving** for JavaScript, CSS, and other resources
- **Security validation** for static assets
- **Configurable data sources** through endpoint options

## Configuration

### Basic Setup

Add a React endpoint to your `turboscript.yml` configuration:

```yaml
endpoints:
  - route: /frontend/*
    method: GET
    path: ./app/frontend
    type: "hybrid"
    options:
      app: "App.html"      # HTML template file
      data: "data"         # Folder containing data endpoints
      assets: "assets"     # Folder containing static assets
```

### Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `app` | HTML template file name | `App.html` | Yes |
| `data` | Folder name containing data endpoints | `api` | No |
| `assets` | Folder name containing static assets | `assets` | No |

## Project Structure

```text
app/frontend/
├── App.html                # HTML template
├── src/                    # React source code
│   ├── App.tsx             # Main React application
│   ├── components/         # React components
│   │   └── Layout.tsx
│   ├── pages/              # Page components
│   │   ├── HomePage.tsx
│   │   ├── SettingsPage.tsx
│   │   └── AboutPage.tsx
│   └── types/
│       └── global.d.ts     # TypeScript global types
├── data/                   # Data endpoints (configurable)
│   ├── index.ts            # Data for /frontend/
│   ├── settings.ts         # Data for /frontend/settings
│   └── about.ts            # Data for /frontend/about
└── assets/                 # Built assets (generated)
    ├── app.js              # Compiled React bundle
    ├── styles.css          # Compiled CSS
    └── app.js.map          # Source map
```

## HTML Template

Create an `App.html` template in your frontend directory:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TurboScript React App</title>
    <link href="/frontend/assets/styles.css" rel="stylesheet">
    <script>
        window.__ROUTE_DATA__ = {
            route: "{{.Route}}",
            data: JSON.parse('{{.Data}}')
        };
    </script>
</head>
<body class="bg-gray-100">
    <div id="root"></div>
    <script src="/frontend/assets/app.js"></script>
</body>
</html>
```

### Template Variables

- `{{.Route}}` - Current route path (e.g., `/frontend/settings`)
- `{{.Data}}` - JSON-encoded data from the data endpoint

## Data Endpoints

Create TypeScript files in your data folder to provide initial data for each route:

### Example: `data/index.ts`

```typescript
export const handle = async (_event: Event): Promise<TurboScriptResponse> => ({
    code: 200,
    response: {
        message: 'Welcome to TurboScript React!',
        user: {
            name: 'John Doe',
            email: 'john@example.com'
        },
        navigation: [
            { label: 'Home', href: '/frontend/' },
            { label: 'Settings', href: '/frontend/settings' },
            { label: 'About', href: '/frontend/about' }
        ]
    }
});
```

### Example: `data/settings.ts`

```typescript
export const handle = async (_event: Event): Promise<TurboScriptResponse> => ({
    code: 200,
    response: {
        message: 'Settings data',
        settings: {
            theme: 'light',
            notifications: true,
            autoSave: false
        },
        timestamp: new Date().toISOString()
    }
});
```

## React Application

### Main App Component

```typescript
import React from 'react';
import { createRoot } from 'react-dom/client';
import Layout from './components/Layout';
import HomePage from './pages/HomePage';
import SettingsPage from './pages/SettingsPage';
import AboutPage from './pages/AboutPage';
import NotFoundPage from './pages/NotFoundPage';
function App() {
    // Get route data injected by hybrid rendering
    const routeData = (window as any).__ROUTE_DATA__ || { route: '/frontend/', data: {} };
    const currentRoute = routeData.route;

    // Determine which page to render based on current route
    let PageComponent;
    switch (currentRoute) {
        case '/frontend/':
            PageComponent = HomePage;
            break;
        case '/frontend/settings':
            PageComponent = SettingsPage;
            break;
        case '/frontend/about':
            PageComponent = AboutPage;
            break;
        default:
            PageComponent = NotFoundPage;
    }

    return (
        <Layout>
            <PageComponent />
        </Layout>
    );
}

// Mount the application
const container = document.getElementById('root');
if (container) {
    const root = createRoot(container);
    root.render(<App />);
}
```

### Accessing Server Data in Components

```typescript
import React from 'react';

interface HomeData {
    message?: string;
    user?: {
        name: string;
        email: string;
    };
    navigation?: Array<{
        label: string;
        href: string;
    }>;
}

export default function HomePage() {
    // Access server-provided data
    const routeData = (window as any).__ROUTE_DATA__;
    const data = routeData?.data as HomeData || {};

    return (
        <div>
            <h1>{data.message}</h1>
            {data.user && (
                <div>
                    <p>Welcome, {data.user.name}!</p>
                    <p>Email: {data.user.email}</p>
                </div>
            )}
            <nav>
                {data.navigation?.map((item, index) => (
                    <a key={index} href={item.href}>{item.label}</a>
                ))}
            </nav>
        </div>
    );
}
```

## Building the Frontend

### Build Script

Create a build script at `scripts/build-frontend.ts`:

```typescript
import * as esbuild from 'esbuild';
import * as fs from 'node:fs';
import * as path from 'node:path';

const frontendDir = './app/frontend';
const outDir = path.join(frontendDir, 'assets');

async function buildReactApp() {
    console.log('🏗️  Building React frontend...');

    // Ensure output directory exists
    if (!fs.existsSync(outDir)) {
        fs.mkdirSync(outDir, { recursive: true });
    }

    // Build React app with esbuild
    await esbuild.build({
        entryPoints: [path.join(frontendDir, 'src/App.tsx')],
        bundle: true,
        outfile: path.join(outDir, 'app.js'),
        format: 'iife',
        target: 'es2020',
        minify: process.env.NODE_ENV === 'production',
        sourcemap: process.env.NODE_ENV !== 'production',
        jsx: 'automatic',
        jsxImportSource: 'react',
        define: {
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
        },
        tsconfig: path.join(frontendDir, 'tsconfig.json'),
    });

    console.log('✅ React frontend built successfully!');
}

buildReactApp().catch(console.error);
```

### Manual Build

```bash
# Build the React frontend
npx tsx scripts/build-frontend.ts

# Or use the Makefile
make build-frontend
```

### Automatic Building

The build system automatically rebuilds the frontend when:

- Using `make dev` (Air hot reloading)
- TypeScript/JSX files change (configured in `.air.toml`)
- Running `make build` or `make build-dist`

## Route Mapping

The React HYBRID system automatically maps routes to data endpoints:

| Frontend Route | Data Endpoint | Description |
|---------------|---------------|-------------|
| `/frontend/` | `data/index.ts` | Homepage data |
| `/frontend/settings` | `data/settings.ts` | Settings page data |
| `/frontend/about` | `data/about.ts` | About page data |
| `/frontend/custom` | `data/custom.ts` | Custom page data |

If a specific data endpoint doesn't exist, the system falls back to `data/index.ts`.

## Static Asset Serving

### Supported File Types

- **JavaScript**: `.js` files with `application/javascript` MIME type
- **CSS**: `.css` files with `text/css` MIME type
- **Images**: `.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.ico`
- **Fonts**: `.woff`, `.woff2`, `.ttf`, `.eot`
- **JSON**: `.json` files with `application/json` MIME type

### Security Features

- **Path validation**: Prevents directory traversal attacks
- **File existence checks**: Returns 404 for missing files
- **MIME type detection**: Proper content types for all assets
- **Cache headers**: Long-term caching for production assets

### Asset URLs

Assets are served under the `/frontend/assets/` path:

```html
<!-- CSS -->
<link href="/frontend/assets/styles.css" rel="stylesheet">

<!-- JavaScript -->
<script src="/frontend/assets/app.js"></script>

<!-- Images -->
<img src="/frontend/assets/logo.png" alt="Logo">
```

## TypeScript Configuration

### Frontend `tsconfig.json`

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["DOM", "DOM.Iterable", "ES6"],
    "allowJs": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "noFallthroughCasesInSwitch": true,
    "module": "ESNext",
    "moduleResolution": "Node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": [
    "src/**/*",
    "**/*.tsx",
    "**/*.ts"
  ],
  "exclude": [
    "node_modules",
    "dist",
    "build"
  ]
}
```

### Global Type Definitions

Create `src/types/global.d.ts`:

```typescript
declare global {
    interface Window {
        __ROUTE_DATA__?: {
            route: string;
            data: any;
        };
    }
}

export {};
```

## Advanced Features

### Authentication Integration

```typescript
// In data endpoints, access authenticated user data
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    // Get authenticated user from event (if using auth middleware)
    const user = event.body?.__user;

    if (!user) {
        return {
            code: 401,
            response: { error: 'Authentication required' }
        };
    }

    return {
        code: 200,
        response: {
            message: `Welcome, ${user.name}!`,
            userProfile: user
        }
    };
};
```

### Database Integration

```typescript
// Use turboQuery in data endpoints
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        const userId = event.pathParameters?.id;

        const [user, posts] = await Promise.all([
            turboQuery('SELECT * FROM users WHERE id = $1', [userId]),
            turboQuery('SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10', [userId])
        ]);

        return {
            code: 200,
            response: {
                user: user[0],
                recentPosts: posts
            }
        };
    } catch (error) {
        return {
            code: 500,
            response: { error: 'Failed to load user data' }
        };
    }
};
```

### Dynamic Routing

For dynamic routes, use path parameters in your data endpoints:

```typescript
// data/user.ts - handles /frontend/user/{id}
export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const userId = event.pathParameters?.id;

    // Load user-specific data
    const userData = await loadUserById(userId);

    return {
        code: 200,
        response: { user: userData }
    };
};
```

## Best Practices

### Performance

1. **Parallel Data Loading**: Use `Promise.all()` in data endpoints for concurrent queries
2. **Asset Optimization**: Minify JavaScript and CSS in production
3. **Caching**: Leverage built-in cache headers for static assets
4. **Bundle Splitting**: Consider code splitting for large applications

### Security

1. **Input Validation**: Always validate data in endpoints before database queries
2. **Authentication**: Use TurboScript's built-in auth utilities
3. **CORS**: Configure CORS headers if serving external domains
4. **Asset Security**: The system automatically prevents directory traversal

### Development

1. **Hot Reloading**: Use `make dev` for automatic rebuilds during development
2. **TypeScript**: Leverage full TypeScript support for type safety
3. **Error Handling**: Implement proper error boundaries in React components
4. **Testing**: Write tests for both data endpoints and React components

## Troubleshooting

### Common Issues

1. **White Screen**: Check browser console for JavaScript errors
2. **404 Assets**: Ensure `make build-frontend` was run successfully
3. **Data Not Loading**: Verify data endpoint paths and return structure
4. **Build Failures**: Check TypeScript configuration and dependencies

### Debug Mode

Enable debug logging in `turboscript.yml`:

```yaml
debug: true
```

This will show detailed logs for:

- Route matching
- Data endpoint execution
- Asset serving
- Template rendering

## Examples

See the complete working example in the `app/frontend/` directory of the TurboScript repository for a full implementation including:

- Multi-page React application
- Navigation with active states
- Form handling
- Data visualization
- Responsive design with Tailwind CSS
