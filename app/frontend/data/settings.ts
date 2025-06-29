import { turboCache } from '@app/utils/turbo-cache';

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    const currentSettings = await getSettings(event);
    const options = getAvailableOptions();

    return {
        code: 200,
        response: {
            settings: currentSettings,
            message: 'Settings retrieved successfully from Redis cache',
            availableThemes: options.themes,
            availableLanguages: options.languages,
            availableTimezones: options.timezones,
            description: 'Configure your application preferences and behavior',
            serverInfo: {
                version: "1.0.0",
                environment: "development",
                dataSource: "redis-server",
                clientIP: getClientIP(event)
            },
            timestamp: new Date().toISOString(),
        },
    };
};

export interface SettingsData {
    theme: string;
    notifications: boolean;
    autoSave: boolean;
    language: string;
    timezone: string;
    emailUpdates: boolean;
    securityNotifications: boolean;
    lastUpdated: string;
    updateCount: number;
}

// Cache key for settings with IP-based isolation
export const SETTINGS_CACHE_KEY = 'user_settings';

// Helper function to get client IP address
function getClientIP(event: Event): string {
    // Try to get real IP from various headers (in order of preference)
    const headers = event.headers;

    // Check for real IP from reverse proxy headers
    const realIP = headers['x-real-ip'] || headers['X-Real-IP'];
    if (realIP && typeof realIP === 'string') {
        return realIP.split(',')[0].trim();
    }

    // Check for forwarded IP
    const forwardedFor = headers['x-forwarded-for'] || headers['X-Forwarded-For'];
    if (forwardedFor && typeof forwardedFor === 'string') {
        return forwardedFor.split(',')[0].trim();
    }

    // Fallback to a default IP for development
    return '127.0.0.1';
}

// Generate IP-based cache key
function getIPBasedCacheKey(event: Event): string {
    const clientIP = getClientIP(event);
    return `${SETTINGS_CACHE_KEY}:${clientIP}`;
}

// Default settings
const defaultSettings: SettingsData = {
    theme: 'light',
    notifications: true,
    autoSave: false,
    language: 'en',
    timezone: 'UTC',
    emailUpdates: true,
    securityNotifications: true,
    lastUpdated: new Date().toISOString(),
    updateCount: 0
};

export async function getSettings(event: Event): Promise<SettingsData> {
    try {
        // Get Redis cache instance using specific driver
        const cache = turboCache('redis-server');

        // Generate IP-based cache key for this client
        const cacheKey = getIPBasedCacheKey(event);

        // Try to get settings from Redis cache
        const cachedSettings = await cache.get(cacheKey);

        if (cachedSettings) {
            // Parse the cached settings if they exist
            const parsed = typeof cachedSettings === 'string'
                ? JSON.parse(cachedSettings) as SettingsData
                : cachedSettings as SettingsData;
            return { ...defaultSettings, ...parsed };
        }

        // If no cached settings, return and cache the defaults
        await cache.set(cacheKey, defaultSettings, 3600); // Cache for 1 hour
        return { ...defaultSettings };
    } catch (_error) {
        // Fallback to default settings if cache fails
        return { ...defaultSettings };
    }
}

function getAvailableOptions(): {
    themes: string[];
    languages: string[];
    timezones: string[];
} {
    return {
        themes: ['light', 'dark', 'auto'],
        languages: ['en', 'es', 'fr', 'de', 'ja'],
        timezones: [
            'UTC',
            'America/New_York',
            'America/Chicago',
            'America/Denver',
            'America/Los_Angeles',
            'Europe/London',
            'Europe/Paris',
            'Asia/Tokyo'
        ]
    };
}
