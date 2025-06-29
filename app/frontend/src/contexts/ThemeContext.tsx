import React, { createContext, useContext, useEffect, useState } from 'react';

interface ThemeContextType {
    theme: string;
    setTheme: (theme: string) => void;
    isDark: boolean;
    isLoading: boolean;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function useTheme() {
    const context = useContext(ThemeContext);
    if (context === undefined) {
        throw new Error('useTheme must be used within a ThemeProvider');
    }
    return context;
}

interface ThemeProviderProps {
    children: React.ReactNode;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
    // Initialize theme from server data
    const routeData = (window as any).__ROUTE_DATA__ || { route: '/frontend', data: {} };
    const fallbackTheme = routeData.data?.settings?.theme || 'light';

    const [theme, setTheme] = useState<string>(fallbackTheme);
    const [systemThemeChange, setSystemThemeChange] = useState(0); // Force re-render counter
    const [isLoading, setIsLoading] = useState(true);

    // Fetch latest settings from server on initialization
    useEffect(() => {
        const fetchSettings = async () => {
            try {
                const response = await fetch('/frontend/data/settings', {
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });

                if (response.ok) {
                    const result = await response.json();

                    // Handle the response format from settings.ts
                    if (result.settings?.theme) {
                        const serverTheme = result.settings.theme;

                        // Always update with server theme to ensure sync
                        setTheme(serverTheme);
                    } else {
                    }
                } else {
                    console.warn('ThemeProvider: Failed to fetch settings, using fallback theme');
                }
            } catch (error) {
                console.error('ThemeProvider: Error fetching settings:', error);
            } finally {
                setIsLoading(false);
            }
        };

        fetchSettings();
    }, []); // Empty dependency array - only run once on mount

    // Calculate if dark mode should be active
    const isDark = theme === 'dark' || (theme === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches);

    // Apply theme changes to the document
    useEffect(() => {
        // Don't apply theme changes while still loading from server
        if (isLoading) return;


        if (isDark) {
            document.documentElement.classList.add('dark');
            document.body.classList.add('dark', 'bg-gray-900');
            document.body.classList.remove('bg-gray-100');
        } else {
            document.documentElement.classList.remove('dark');
            document.body.classList.remove('dark', 'bg-gray-900');
            document.body.classList.add('bg-gray-100');
        }
    }, [theme, isDark, systemThemeChange, isLoading]);

    // Listen for system theme changes when theme is 'auto'
    useEffect(() => {
        if (theme === 'auto') {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            const handleChange = () => {
                // Force re-render by incrementing counter instead of changing theme state
                setSystemThemeChange(prev => prev + 1);
            };

            mediaQuery.addEventListener('change', handleChange);
            return () => mediaQuery.removeEventListener('change', handleChange);
        }
    }, [theme]);

    return (
        <ThemeContext.Provider value={{ theme, setTheme, isDark, isLoading }}>
            {children}
        </ThemeContext.Provider>
    );
}
