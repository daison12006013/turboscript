import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useHybridDataWithNavigation } from '../hooks/useHybridNavigation';

interface SettingsData {
    message?: string;
    settings?: {
        theme?: string;
        notifications?: boolean;
        autoSave?: boolean;
        language?: string;
        timezone?: string;
        emailUpdates?: boolean;
        securityNotifications?: boolean;
    };
    availableThemes?: string[];
    availableLanguages?: string[];
    availableTimezones?: string[];
    description?: string;
    serverInfo?: {
        version?: string;
        environment?: string;
        dataSource?: string;
        clientIP?: string;
    };
    timestamp?: string;
    error?: string;
}

export default function SettingsPage() {
    // Get theme context for global theme management
    const { theme: globalTheme, setTheme: setGlobalTheme, isDark } = useTheme();

    // Use HYBRID data hook to fetch data from /hybrid/data/settings endpoint
    const { data, loading, error } = useHybridDataWithNavigation<SettingsData>(
        '/hybrid/data/settings',
        '/hybrid/settings'
    );

    // Initialize settings from server data (HYBRID) - prioritize server data over global theme
    const [settings, setSettings] = useState({
        theme: data?.settings?.theme || globalTheme || 'light',
        notifications: data?.settings?.notifications ?? true,
        autoSave: data?.settings?.autoSave ?? false,
        language: data?.settings?.language || 'en',
        timezone: data?.settings?.timezone || 'UTC',
        emailUpdates: data?.settings?.emailUpdates ?? true,
        securityNotifications: data?.settings?.securityNotifications ?? true,
    });

    // Update settings when data changes (from navigation or refetch)
    useEffect(() => {
        if (data?.settings) {
            setSettings({
                theme: data.settings.theme || globalTheme || 'light',
                notifications: data.settings.notifications ?? true,
                autoSave: data.settings.autoSave ?? false,
                language: data.settings.language || 'en',
                timezone: data.settings.timezone || 'UTC',
                emailUpdates: data.settings.emailUpdates ?? true,
                securityNotifications: data.settings.securityNotifications ?? true,
            });
        }
    }, [data, globalTheme]);

    const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
    const [lastResponse, setLastResponse] = useState<any>(null);

    // Get theme-aware CSS classes using the actual theme state
    const getThemeClasses = () => {
        return {
            background: `${isDark ? 'bg-gray-900' : 'bg-gray-100'} min-h-screen transition-colors duration-300`,
            card: `${isDark ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-100'}`,
            text: {
                primary: isDark ? 'text-white' : 'text-gray-900',
                secondary: isDark ? 'text-gray-300' : 'text-gray-600',
                muted: isDark ? 'text-gray-400' : 'text-gray-500',
            },
            input: `${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300 text-gray-900'}`,
            gradient: 'bg-gradient-to-r from-blue-600 to-purple-600',
            button: {
                primary: isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-600 hover:bg-blue-700',
                success: isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-600 hover:bg-green-700',
                error: isDark ? 'bg-red-600 hover:bg-red-700' : 'bg-red-600 hover:bg-red-700',
            },
            border: isDark ? 'border-gray-700' : 'border-gray-100',
            codeBlock: `${isDark ? 'bg-gray-800 border-gray-700 text-gray-300' : 'bg-gray-50 border-gray-200 text-gray-700'}`,
        };
    };

    const theme = getThemeClasses();

    const handleSave = async () => {
        setSaveStatus('saving');

        try {
            // Make real API call to update settings
            const response = await fetch('/hybrid/data/settings/update', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    theme: settings.theme,
                    notifications: settings.notifications,
                    autoSave: settings.autoSave,
                    language: settings.language,
                    timezone: settings.timezone,
                    emailUpdates: settings.emailUpdates,
                    securityNotifications: settings.securityNotifications,
                }),
            });

            const result = await response.json();
            setLastResponse(result);

            if (response.ok && result.status === 'success') {
                setSaveStatus('saved');

                // Update local settings with the server response to ensure sync
                if (result.data?.settings) {
                    const updatedSettings = {
                        theme: result.data.settings.theme || settings.theme,
                        notifications: result.data.settings.notifications ?? settings.notifications,
                        autoSave: result.data.settings.autoSave ?? settings.autoSave,
                        language: result.data.settings.language || settings.language,
                        timezone: result.data.settings.timezone || settings.timezone,
                        emailUpdates: result.data.settings.emailUpdates ?? settings.emailUpdates,
                        securityNotifications: result.data.settings.securityNotifications ?? settings.securityNotifications,
                    };

                    setSettings(updatedSettings);

                    // Update global theme context with the saved theme
                    setGlobalTheme(updatedSettings.theme);
                }

                // Show success message and reset after 3 seconds
                setTimeout(() => setSaveStatus('idle'), 3000);

            } else {
                setSaveStatus('error');
                console.error('Failed to update settings:', result);

                // Reset to idle after 3 seconds
                setTimeout(() => setSaveStatus('idle'), 3000);
            }
        } catch (error) {
            setSaveStatus('error');
            console.error('Error updating settings:', error);

            // Reset to idle after 3 seconds
            setTimeout(() => setSaveStatus('idle'), 3000);
        }
    };

    const updateSetting = (key: string, value: any) => {
        setSettings(prev => ({ ...prev, [key]: value }));

        // Immediately update global theme context when theme changes
        if (key === 'theme') {
            setGlobalTheme(value);
        }
    };

    return (
        <div className={`${theme.background} transition-colors duration-300`}>
            <div className="space-y-8 p-4">
                {/* Loading State */}
                {loading && (
                    <div className="text-center py-8">
                        <div className="inline-flex items-center px-4 py-2 font-semibold leading-6 text-sm shadow rounded-md text-white bg-blue-500 hover:bg-blue-400 transition ease-in-out duration-150 cursor-not-allowed">
                            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                            </svg>
                            Loading settings from server...
                        </div>
                    </div>
                )}

                {/* Error State */}
                {error && (
                    <div className="bg-red-50 border border-red-200 rounded-md p-4">
                        <div className="flex">
                            <div className="flex-shrink-0">
                                <svg className="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-3">
                                <h3 className="text-sm font-medium text-red-800">Error loading settings</h3>
                                <div className="mt-2 text-sm text-red-700">
                                    <p>{error}</p>
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {/* Header */}
                <div className={`${theme.card} rounded-xl shadow-lg border overflow-hidden`}>
                    <div className={`${theme.gradient} px-8 py-6`}>
                        <h2 className="text-3xl font-bold text-white flex items-center">
                            <svg className="w-8 h-8 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            </svg>
                            Settings & Preferences
                        </h2>
                        <p className="text-blue-100 mt-2">Customize your TurboScript experience</p>
                    </div>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Settings Form */}
                    <div className="lg:col-span-2 space-y-6">
                        {/* Appearance Settings */}
                        <div className={`${theme.card} rounded-xl shadow-lg border`}>
                            <div className={`px-6 py-4 border-b ${theme.border}`}>
                                <h3 className={`text-lg font-semibold ${theme.text.primary} flex items-center`}>
                                    <svg className="w-5 h-5 text-purple-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zM21 5a2 2 0 00-2-2h-4a2 2 0 00-2 2v12a4 4 0 004 4h4a2 2 0 002-2V5z" />
                                    </svg>
                                    Appearance
                                </h3>
                            </div>
                            <div className="p-6 space-y-6">
                                {/* Theme Setting */}
                                <div>
                                    <label className={`text-base font-medium ${theme.text.primary} mb-3 block`}>Theme Preference</label>
                                    <div className="grid grid-cols-3 gap-3">{[
                                        { value: 'light', label: 'Light', icon: '☀️' },
                                        { value: 'dark', label: 'Dark', icon: '🌙' },
                                        { value: 'auto', label: 'Auto', icon: '🔄' }
                                    ].map((option) => (
                                        <label
                                            key={option.value}
                                            className={`${settings.theme === option.value
                                                ? `${isDark ? 'bg-blue-900 border-blue-600 text-blue-300' : 'bg-blue-50 border-blue-300 text-blue-700'}`
                                                : `${theme.card} ${theme.border} ${theme.text.primary} hover:opacity-80`
                                                } relative block w-full border-2 rounded-lg p-4 cursor-pointer focus:outline-none transition-all`}
                                        >
                                            <input
                                                type="radio"
                                                name="theme"
                                                value={option.value}
                                                checked={settings.theme === option.value}
                                                onChange={(e) => updateSetting('theme', e.target.value)}
                                                className="sr-only"
                                            />
                                            <div className="text-center">
                                                <div className="text-2xl mb-2">{option.icon}</div>
                                                <div className="text-sm font-medium">{option.label}</div>
                                            </div>
                                        </label>
                                    ))}
                                    </div>
                                </div>

                                {/* Language Setting */}
                                <div>
                                    <label htmlFor="language" className={`text-base font-medium ${theme.text.primary} mb-3 block`}>
                                        Language
                                    </label>
                                    <select
                                        id="language"
                                        value={settings.language}
                                        onChange={(e) => updateSetting('language', e.target.value)}
                                        className={`block w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${theme.input}`}
                                    >
                                        <option value="en">English</option>
                                        <option value="es">Español</option>
                                        <option value="fr">Français</option>
                                        <option value="de">Deutsch</option>
                                        <option value="ja">日本語</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        {/* Notifications Settings */}
                        <div className={`${theme.card} rounded-xl shadow-lg border ${theme.border}`}>
                            <div className={`px-6 py-4 border-b ${theme.border}`}>
                                <h3 className={`text-lg font-semibold ${theme.text.primary} flex items-center`}>
                                    <svg className="w-5 h-5 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-5 5-5-5h5V3h5v14z" />
                                    </svg>
                                    Notifications
                                </h3>
                            </div>
                            <div className="p-6 space-y-6">
                                {[
                                    {
                                        key: 'notifications',
                                        title: 'Browser Notifications',
                                        description: 'Get notified when important events occur in your applications',
                                        icon: '🔔'
                                    },
                                    {
                                        key: 'emailUpdates',
                                        title: 'Email Updates',
                                        description: 'Receive periodic updates about new features and improvements',
                                        icon: '📧'
                                    },
                                    {
                                        key: 'securityNotifications',
                                        title: 'Security Alerts',
                                        description: 'Get notified about security-related events and updates',
                                        icon: '🔐'
                                    }
                                ].map((item) => (
                                    <div key={item.key} className={`flex items-start justify-between p-4 ${theme.codeBlock} rounded-lg`}>
                                        <div className="flex items-start space-x-3">
                                            <div className="text-2xl">{item.icon}</div>
                                            <div>
                                                <h4 className={`text-sm font-medium ${theme.text.primary}`}>{item.title}</h4>
                                                <p className={`text-sm ${theme.text.secondary} mt-1`}>{item.description}</p>
                                            </div>
                                        </div>
                                        <label className="relative inline-flex items-center cursor-pointer">
                                            <input
                                                type="checkbox"
                                                checked={settings[item.key as keyof typeof settings] as boolean}
                                                onChange={(e) => updateSetting(item.key, e.target.checked)}
                                                className="sr-only peer"
                                            />
                                            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                                        </label>
                                    </div>
                                ))}
                            </div>
                        </div>

                        {/* Advanced Settings */}
                        <div className={`${theme.card} rounded-xl shadow-lg border ${theme.border}`}>
                            <div className={`px-6 py-4 border-b ${theme.border}`}>
                                <h3 className={`text-lg font-semibold ${theme.text.primary} flex items-center`}>
                                    <svg className="w-5 h-5 text-orange-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
                                    </svg>
                                    Advanced
                                </h3>
                            </div>
                            <div className="p-6 space-y-6">
                                <div className={`flex items-center justify-between p-4 ${theme.codeBlock} rounded-lg`}>
                                    <div className="flex items-center space-x-3">
                                        <div className="text-2xl">💾</div>
                                        <div>
                                            <h4 className={`text-sm font-medium ${theme.text.primary}`}>Auto-save Changes</h4>
                                            <p className={`text-sm ${theme.text.secondary} mt-1`}>Automatically save your changes as you work</p>
                                        </div>
                                    </div>
                                    <label className="relative inline-flex items-center cursor-pointer">
                                        <input
                                            type="checkbox"
                                            checked={settings.autoSave}
                                            onChange={(e) => updateSetting('autoSave', e.target.checked)}
                                            className="sr-only peer"
                                        />
                                        <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                                    </label>
                                </div>

                                <div>
                                    <label htmlFor="timezone" className={`text-base font-medium ${theme.text.primary} mb-3 block`}>
                                        Timezone
                                    </label>
                                    <select
                                        id="timezone"
                                        value={settings.timezone}
                                        onChange={(e) => updateSetting('timezone', e.target.value)}
                                        className={`block w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${theme.input}`}
                                    >
                                        <option value="UTC">UTC</option>
                                        <option value="America/New_York">Eastern Time</option>
                                        <option value="America/Chicago">Central Time</option>
                                        <option value="America/Denver">Mountain Time</option>
                                        <option value="America/Los_Angeles">Pacific Time</option>
                                        <option value="Europe/London">London</option>
                                        <option value="Europe/Paris">Paris</option>
                                        <option value="Asia/Tokyo">Tokyo</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        {/* Save Button */}
                        <div className="flex justify-end">
                            <button
                                onClick={handleSave}
                                disabled={saveStatus === 'saving'}
                                className={`inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-lg shadow-sm text-white transition-all duration-200 ${saveStatus === 'saving'
                                    ? 'bg-gray-400 cursor-not-allowed'
                                    : saveStatus === 'saved'
                                        ? `${theme.button.success}`
                                        : saveStatus === 'error'
                                            ? `${theme.button.error}`
                                            : `${theme.button.primary} focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500`
                                    }`}
                            >
                                {saveStatus === 'saving' && (
                                    <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                    </svg>
                                )}
                                {saveStatus === 'saved' && (
                                    <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                    </svg>
                                )}
                                {saveStatus === 'error' && (
                                    <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                    </svg>
                                )}
                                {saveStatus === 'saving' ? 'Saving...' : saveStatus === 'saved' ? 'Saved!' : saveStatus === 'error' ? 'Error - Try Again' : 'Save Settings'}
                            </button>
                        </div>
                    </div>

                    {/* Sidebar Info */}
                    <div className="space-y-6">
                        {/* Current Settings Preview */}
                        <div className={`${theme.card} rounded-xl shadow-lg border`}>
                            <div className={`px-6 py-4 border-b ${theme.border}`}>
                                <h3 className={`text-lg font-semibold ${theme.text.primary}`}>Current Settings</h3>
                            </div>
                            <div className="p-6">
                                <pre className={`text-sm ${theme.codeBlock} p-4 rounded-lg border overflow-x-auto font-mono`}>
                                    {JSON.stringify(settings, null, 2)}
                                </pre>
                            </div>
                        </div>

                        {/* Server Data */}
                        {data && Object.keys(data).length > 0 && (
                            <div className={`${theme.card} rounded-xl shadow-lg border ${theme.border}`}>
                                <div className={`px-6 py-4 border-b ${theme.border}`}>
                                    <h3 className={`text-lg font-semibold ${theme.text.primary} flex items-center`}>
                                        <svg className="w-5 h-5 text-blue-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                                        </svg>
                                        Initial Server Data (HYBRID)
                                    </h3>
                                    <p className={`text-sm ${theme.text.secondary} mt-1`}>Data loaded from server on page render</p>
                                </div>
                                <div className="p-6">
                                    <pre className={`text-sm ${theme.codeBlock} p-4 rounded-lg border overflow-x-auto font-mono`}>
                                        {JSON.stringify(data, null, 2)}
                                    </pre>
                                </div>
                            </div>
                        )}

                        {/* Last API Response */}
                        {lastResponse && (
                            <div className={`${theme.card} rounded-xl shadow-lg border ${theme.border}`}>
                                <div className={`px-6 py-4 border-b ${theme.border}`}>
                                    <h3 className={`text-lg font-semibold ${theme.text.primary} flex items-center`}>
                                        <svg className="w-5 h-5 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                                        </svg>
                                        Last API Response
                                    </h3>
                                    <p className={`text-sm ${theme.text.secondary} mt-1`}>Real-time response from the update endpoint</p>
                                </div>
                                <div className="p-6">
                                    <div className="mb-4">
                                        <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${lastResponse.status === 'success'
                                            ? `${isDark ? 'bg-green-900 text-green-300' : 'bg-green-100 text-green-800'}`
                                            : `${isDark ? 'bg-red-900 text-red-300' : 'bg-red-100 text-red-800'}`
                                            }`}>
                                            {lastResponse.status === 'success' && (
                                                <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                                </svg>
                                            )}
                                            {lastResponse.status === 'error' && (
                                                <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                                </svg>
                                            )}
                                            {lastResponse.status}
                                        </span>
                                    </div>
                                    <pre className={`text-sm ${theme.codeBlock} p-4 rounded-lg border overflow-x-auto font-mono`}>
                                        {JSON.stringify(lastResponse, null, 2)}
                                    </pre>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
