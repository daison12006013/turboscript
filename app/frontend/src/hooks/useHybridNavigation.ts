import { useState, useEffect, useCallback } from 'react';

// Global window interface extensions for HYBRID data
declare global {
    interface Window {
        __ROUTE_DATA__?: {
            route: string;
            data: unknown;
        };
        __NAVIGATION_DATA__?: {
            route: string;
            data: unknown;
        };
    }
}

export interface HybridDataState<T = unknown> {
    data: T | null;
    loading: boolean;
    error: string | null;
    refetch: () => Promise<void>;
}

interface TurboScriptResponse {
    response?: unknown;
    data?: unknown;
    code?: number;
    message?: string;
}

interface NavigationHook {
    navigateWithData: (
        routePath: string,
        dataEndpoint: string,
        navigate: (path: string) => void
    ) => Promise<void>;
    isNavigating: boolean;
}

// Browser environment check
function isBrowser(): boolean {
    return typeof window !== 'undefined';
}

// Safe window access
function getWindow(): Window | null {
    return isBrowser() ? globalThis.window : null;
}

/**
 * Hook for managing HYBRID data fetching with React Router navigation
 * Handles both initial HYBRID data and subsequent client-side navigation
 */
export function useHybridData<T = unknown>(dataEndpoint: string, routePath: string): HybridDataState<T> {
    const [data, setData] = useState<T | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Check if we have initial HYBRID data
    const getInitialData = useCallback((): T | null => {
        try {
            const win = getWindow();
            if (!win) return null;

            const routeData = win.__ROUTE_DATA__;

            // If we're on the expected route and have HYBRID data, use it
            if (routeData && routeData.route === routePath && routeData.data) {
                return routeData.data as T;
            }

            return null;
        } catch (_err) {
            // Failed to parse initial HYBRID data - silently handle the error
            return null;
        }
    }, [routePath]);

    // Fetch data from the server endpoint
    const fetchData = useCallback(async (): Promise<void> => {
        setLoading(true);
        setError(null);

        try {
            const response = await globalThis.fetch(dataEndpoint, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json',
                    // Add header to indicate this is an API request (for your isAPIRequest logic)
                    'X-Requested-With': 'XMLHttpRequest'
                },
                credentials: 'same-origin'
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const responseData = await response.json() as TurboScriptResponse;

            // Handle different response structures from your TurboScript endpoints
            let extractedData: T;

            if (responseData.response) {
                // Standard TurboScript response format: { code: 200, response: {...} }
                extractedData = responseData.response as T;
            } else if (responseData.data) {
                // Alternative response format: { data: {...} }
                extractedData = responseData.data as T;
            } else {
                // Direct response data
                extractedData = responseData as T;
            }

            setData(extractedData);
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to fetch data';
            setError(errorMessage);
            console.error('HYBRID Data fetch error:', err);
        } finally {
            setLoading(false);
        }
    }, [dataEndpoint]);

    // Initialize data on mount
    useEffect(() => {
        const initialData = getInitialData();

        if (initialData) {
            // Use HYBRID data if available
            setData(initialData);
            setLoading(false);
        } else {
            // Fetch data if no HYBRID data available
            void fetchData();
        }
    }, [getInitialData, fetchData]);

    // Provide refetch capability for manual refresh
    const refetch = useCallback(async (): Promise<void> => {
        await fetchData();
    }, [fetchData]);

    return {
        data,
        loading,
        error,
        refetch
    };
}

/**
 * Hook for navigation with HYBRID data fetching
 * This provides the core functionality for HYBRID-aware navigation
 */
export function useHybridNavigation(): NavigationHook {
    const [isNavigating, setIsNavigating] = useState(false);

    const navigateWithData = useCallback(async (
        routePath: string,
        dataEndpoint: string,
        navigate: (path: string) => void
    ): Promise<void> => {
        setIsNavigating(true);

        try {
            // Pre-fetch data for the target route
            const response = await globalThis.fetch(dataEndpoint, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                credentials: 'same-origin'
            });

            if (!response.ok) {
                throw new Error(`Failed to fetch data for ${routePath}`);
            }

            const responseData = await response.json() as TurboScriptResponse;

            // Store the fetched data for the target route
            // This will be picked up by the useHybridData hook on the target page
            const targetRouteData = {
                route: routePath,
                data: responseData.response ?? responseData.data ?? responseData
            };

            // Temporarily store in a global variable for the target component
            const win = getWindow();
            if (win) {
                win.__NAVIGATION_DATA__ = targetRouteData;
            }

            // Perform the navigation
            navigate(routePath);

        } catch (err) {
            console.error('Navigation with data fetch failed:', err);
            // Navigate anyway, let the target page handle the error
            navigate(routePath);
        } finally {
            setIsNavigating(false);
        }
    }, []);

    return {
        navigateWithData,
        isNavigating
    };
}

/**
 * Enhanced version of useHybridData that also checks for navigation data
 */
export function useHybridDataWithNavigation<T = unknown>(dataEndpoint: string, routePath: string): HybridDataState<T> {
    const [data, setData] = useState<T | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Check for both HYBRID data and navigation data
    const getAvailableData = useCallback((): T | null => {
        try {
            const win = getWindow();
            if (!win) return null;

            // First check for navigation data (from HYBRID navigation)
            const navigationData = win.__NAVIGATION_DATA__;
            if (navigationData && navigationData.route === routePath) {
                // Clear the navigation data after use
                delete win.__NAVIGATION_DATA__;
                return navigationData.data as T;
            }

            // Fall back to initial HYBRID data
            const routeData = win.__ROUTE_DATA__;
            if (routeData && routeData.route === routePath && routeData.data) {
                return routeData.data as T;
            }

            return null;
        } catch (err) {
            console.warn('Failed to parse available data:', err);
            return null;
        }
    }, [routePath]);

    // Fetch data from the server endpoint
    const fetchData = useCallback(async (): Promise<void> => {
        setLoading(true);
        setError(null);

        try {
            const response = await globalThis.fetch(dataEndpoint, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                credentials: 'same-origin'
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const responseData = await response.json() as TurboScriptResponse;

            let extractedData: T;
            if (responseData.response) {
                extractedData = responseData.response as T;
            } else if (responseData.data) {
                extractedData = responseData.data as T;
            } else {
                extractedData = responseData as T;
            }

            setData(extractedData);
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to fetch data';
            setError(errorMessage);
            console.error('HYBRID Data fetch error:', err);
        } finally {
            setLoading(false);
        }
    }, [dataEndpoint]);

    // Initialize data on mount
    useEffect(() => {
        const availableData = getAvailableData();

        if (availableData) {
            setData(availableData);
            setLoading(false);
        } else {
            void fetchData();
        }
    }, [getAvailableData, fetchData]);

    // Provide refetch capability
    const refetch = useCallback(async (): Promise<void> => {
        await fetchData();
    }, [fetchData]);

    return {
        data,
        loading,
        error,
        refetch
    };
}
