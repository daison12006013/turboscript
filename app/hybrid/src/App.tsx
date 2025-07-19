import React, { useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter, RouterProvider, Outlet, useNavigate, useLocation } from 'react-router-dom';
import { ThemeProvider } from './contexts/ThemeContext';
import Layout from './components/Layout';
import HomePage from './pages/HomePage';
import SettingsPage from './pages/SettingsPage';
import AboutPage from './pages/AboutPage';
import NotFoundPage from './pages/NotFoundPage';

// Get the current route data from hybrid rendering
const routeData = (window as any).__ROUTE_DATA__ || { route: '/hybrid', data: {} };

// Layout wrapper that provides the Outlet for nested routes and handles HYBRID navigation
function LayoutWrapper() {
    const navigate = useNavigate();
    const location = useLocation();

    useEffect(() => {
        // Convert server route to client route
        const serverRoute = routeData.route;
        const expectedClientPath = serverRoute.replace('/hybrid', '') || '/';

        // If we're not already on the expected path, navigate to it
        // This handles cases where HYBRID determines a different route than what's in the URL
        if (location.pathname !== expectedClientPath) {
            // Uncomment the line below if you want automatic navigation based on HYBRID
            // navigate(expectedClientPath, { replace: true });
        }
    }, [navigate, location]);

    return (
        <Layout>
            <Outlet />
        </Layout>
    );
}

// Create the router with data-driven routing
const router = createBrowserRouter([
    {
        path: "/",
        element: <LayoutWrapper />,
        children: [
            {
                path: "/hybrid",
                element: <HomePage />
            },
            {
                path: "/hybrid/settings",
                element: <SettingsPage />
            },
            {
                path: "/hybrid/about",
                element: <AboutPage />
            },
            {
                path: "/hybrid/*",
                element: <NotFoundPage />
            }
        ]
    }
], {
    basename: "/"
});

function App() {
    return (
        <ThemeProvider>
            <RouterProvider router={router} />
        </ThemeProvider>
    );
}

// Mount the app
const container = document.getElementById('root');
if (container) {
    const root = createRoot(container);
    root.render(<App />);
} else {
    console.error('Root container not found');
}
