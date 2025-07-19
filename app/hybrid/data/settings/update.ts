import { turboCache } from '@app/utils/turbo-cache';
import type { SettingsData } from '../settings';
import { getSettings, SETTINGS_CACHE_KEY } from '../settings';

interface SettingsUpdateBody {
    theme?: string;
    notifications?: boolean;
    autoSave?: boolean;
    language?: string;
    timezone?: string;
    emailUpdates?: boolean;
    securityNotifications?: boolean;
}

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

async function updateSettings(updates: Partial<SettingsData>, event: Event): Promise<SettingsData> {
    // Get Redis cache instance using specific driver
    const cache = turboCache('redis-server');

    // Generate IP-based cache key for this client
    const cacheKey = getIPBasedCacheKey(event);

    const currentSettings = await getSettings(event);

    const updatedSettings: SettingsData = {
        ...currentSettings,
        ...updates,
        lastUpdated: new Date().toISOString(),
        updateCount: currentSettings.updateCount + 1
    };

    await cache.set(cacheKey, updatedSettings, 3600);
    return { ...updatedSettings };
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

export const handle = async (event: Event): Promise<TurboScriptResponse> => {
    try {
        // Simulate a small delay to show real backend processing
        // Using Date-based delay instead of setTimeout for TurboScript compatibility
        const startTime = Date.now();
        while (Date.now() - startTime < 250) {
            // Small delay to demonstrate backend processing
        }

        // Get the request body
        const body = event.body as SettingsUpdateBody | undefined;

        if (!body || typeof body !== 'object') {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "Request body is required"
                }
            };
        }

        const options = getAvailableOptions();

        // Validate theme setting
        if (body.theme && !options.themes.includes(body.theme)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: `Invalid theme. Must be one of: ${options.themes.join(', ')}`
                }
            };
        }

        // Validate language setting
        if (body.language && !options.languages.includes(body.language)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: `Invalid language. Must be one of: ${options.languages.join(', ')}`
                }
            };
        }

        // Validate timezone setting
        if (body.timezone && !options.timezones.includes(body.timezone)) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: `Invalid timezone. Must be one of: ${options.timezones.join(', ')}`
                }
            };
        }

        // Validate and update settings
        const validSettings = ['theme', 'notifications', 'autoSave', 'language', 'timezone', 'emailUpdates', 'securityNotifications'] as const;
        const updates: Record<string, unknown> = {};

        for (const key of validSettings) {
            if (body[key] !== undefined) {
                updates[key] = body[key];
            }
        }

        if (Object.keys(updates).length === 0) {
            return {
                code: 400,
                response: {
                    status: "error",
                    message: "No valid settings provided"
                }
            };
        }

        // Update the settings using the shared store
        const updatedSettings = await updateSettings(updates, event);

        // Return success response with updated settings
        return {
            code: 200,
            response: {
                status: "success",
                message: `Settings updated successfully (update #${updatedSettings.updateCount})`,
                data: {
                    settings: updatedSettings,
                    updatedFields: Object.keys(updates),
                    serverTimestamp: new Date().toISOString(),
                    // Show it's a real backend by including server info
                    serverInfo: {
                        version: "1.0.0",
                        environment: "development",
                        processingTime: "250ms",
                        dataSource: "redis-server",
                        clientIP: getClientIP(event)
                    }
                }
            }
        };
    } catch (_error) {
        return {
            code: 500,
            response: {
                status: "error",
                message: _error instanceof Error ? _error.message : "An unexpected error occurred",
                timestamp: new Date().toISOString()
            }
        };
    }
};
