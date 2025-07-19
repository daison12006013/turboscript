import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useHybridDataWithNavigation } from '../hooks/useHybridNavigation';
import { HybridLink } from '../components/HybridLink';

interface HomeData {
    message?: string;
    error?: string;
    framework?: string;
    version?: string;
    description?: string;
    features?: string[];
    timestamp?: string;
    stats?: {
        performance?: string;
        responseTime?: string;
        memoryUsage?: string;
        concurrentUsers?: string;
        activeUsers?: number;
        requestsProcessed?: number;
        uptime?: string;
    };
}

export default function HomePage() {
    const [stats, setStats] = useState({ requestsProcessed: 0, activeUsers: 0 });

    // Use HYBRID data hook to fetch data from /hybrid/data endpoint
    const { data, loading, error } = useHybridDataWithNavigation<HomeData>(
        '/hybrid/data',
        '/hybrid'
    );

    // Animate counters
    useEffect(() => {
        if (!data) return;

        const animateCounter = (target: number, setter: (value: number) => void) => {
            let current = 0;
            const increment = target / 50;
            const timer = setInterval(() => {
                current += increment;
                if (current >= target) {
                    setter(target);
                    clearInterval(timer);
                } else {
                    setter(Math.floor(current));
                }
            }, 30);
        };

        animateCounter(data.stats?.requestsProcessed || 12847, (val) =>
            setStats(prev => ({ ...prev, requestsProcessed: val }))
        );
        animateCounter(data.stats?.activeUsers || 256, (val) =>
            setStats(prev => ({ ...prev, activeUsers: val }))
        );
    }, [data]);

    return (
        <div className="space-y-8">
            {/* Loading State */}
            {loading && (
                <div className="text-center py-8">
                    <div className="inline-flex items-center px-4 py-2 font-semibold leading-6 text-sm shadow rounded-md text-white bg-blue-500 hover:bg-blue-400 transition ease-in-out duration-150 cursor-not-allowed">
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Loading data from server...
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
                            <h3 className="text-sm font-medium text-red-800">Error loading data</h3>
                            <div className="mt-2 text-sm text-red-700">
                                <p>{error}</p>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Hero Section */}
            <div className="relative bg-gradient-to-br from-blue-600 via-purple-600 to-blue-800 rounded-2xl overflow-hidden">
                <div className="absolute inset-0 bg-black/20"></div>
                <div className="relative px-8 py-12 sm:px-12 sm:py-16">
                    <div className="max-w-3xl">
                        <h1 className="text-4xl sm:text-5xl font-bold text-white mb-6">
                            Welcome to{' '}
                            <span className="bg-gradient-to-r from-yellow-300 to-orange-300 bg-clip-text text-transparent">
                                TurboScript
                            </span>
                        </h1>
                        <p className="text-xl text-blue-100 mb-8 leading-relaxed">
                            Experience the future of web development with our hybrid TypeScript/Go architecture.
                            Build lightning-fast applications with unmatched developer experience.
                        </p>
                        <div className="flex flex-col sm:flex-row gap-4">
                            <HybridLink
                                to="/hybrid/about"
                                dataEndpoint="/hybrid/data/about"
                                className="inline-flex items-center justify-center px-6 py-3 bg-white text-blue-600 font-semibold rounded-lg hover:bg-blue-50 transition-all duration-200 shadow-lg hover:shadow-xl group"
                            >
                                Learn More
                                <svg className="ml-2 w-5 h-5 group-hover:translate-x-1 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                                </svg>
                            </HybridLink>
                            <HybridLink
                                to="/hybrid/settings"
                                dataEndpoint="/hybrid/data/settings"
                                className="inline-flex items-center justify-center px-6 py-3 bg-blue-500/20 text-white font-semibold rounded-lg hover:bg-blue-500/30 transition-all duration-200 border border-white/20 backdrop-blur-sm"
                            >
                                Configure Settings
                            </HybridLink>
                        </div>
                    </div>
                </div>

                {/* Floating elements */}
                <div className="absolute top-8 right-8 w-16 h-16 bg-white/10 rounded-full animate-pulse"></div>
                <div className="absolute bottom-8 right-16 w-8 h-8 bg-yellow-300/30 rounded-full animate-bounce"></div>
            </div>

            {/* Stats Section */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-100 dark:border-gray-700 hover:shadow-xl transition-shadow">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="w-12 h-12 bg-gradient-to-br from-green-400 to-green-600 rounded-lg flex items-center justify-center">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Requests Processed</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.requestsProcessed.toLocaleString()}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-100 dark:border-gray-700 hover:shadow-xl transition-shadow">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="w-12 h-12 bg-gradient-to-br from-blue-400 to-blue-600 rounded-lg flex items-center justify-center">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Active Users</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.activeUsers.toLocaleString()}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-100 dark:border-gray-700 hover:shadow-xl transition-shadow">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="w-12 h-12 bg-gradient-to-br from-purple-400 to-purple-600 rounded-lg flex items-center justify-center">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{data?.stats?.uptime || '99.9%'}</p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Features Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg overflow-hidden border border-gray-100 dark:border-gray-700">
                    <div className="p-8">
                        <div className="flex items-center mb-6">
                            <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-700 rounded-lg flex items-center justify-center mr-4">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                                </svg>
                            </div>
                            <h3 className="text-xl font-bold text-gray-900 dark:text-white">Lightning Performance</h3>
                        </div>
                        <p className="text-gray-600 dark:text-gray-300 mb-4">
                            Hybrid TypeScript/Go architecture delivers exceptional speed with Go's runtime
                            performance and TypeScript's developer productivity.
                        </p>
                        <div className="flex items-center text-sm text-blue-600 dark:text-blue-400 font-medium">
                            <span>Learn about our architecture</span>
                            <svg className="ml-1 w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                            </svg>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg overflow-hidden border border-gray-100 dark:border-gray-700">
                    <div className="p-8">
                        <div className="flex items-center mb-6">
                            <div className="w-12 h-12 bg-gradient-to-br from-green-500 to-green-700 rounded-lg flex items-center justify-center mr-4">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                            <h3 className="text-xl font-bold text-gray-900 dark:text-white">Type Safety</h3>
                        </div>
                        <p className="text-gray-600 dark:text-gray-300 mb-4">
                            Full TypeScript support across the entire stack ensures robust code quality
                            and excellent developer experience with intelligent autocomplete.
                        </p>
                        <div className="flex items-center text-sm text-green-600 dark:text-green-400 font-medium">
                            <span>Explore type definitions</span>
                            <svg className="ml-1 w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                            </svg>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg overflow-hidden border border-gray-100 dark:border-gray-700">
                    <div className="p-8">
                        <div className="flex items-center mb-6">
                            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-700 rounded-lg flex items-center justify-center mr-4">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                                </svg>
                            </div>
                            <h3 className="text-xl font-bold text-gray-900 dark:text-white">Developer Experience</h3>
                        </div>
                        <p className="text-gray-600 dark:text-gray-300 mb-4">
                            Hot reloading, intuitive API design, and comprehensive tooling make development
                            fast and enjoyable with real-time feedback.
                        </p>
                        <div className="flex items-center text-sm text-purple-600 dark:text-purple-400 font-medium">
                            <span>See development tools</span>
                            <svg className="ml-1 w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                            </svg>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg overflow-hidden border border-gray-100 dark:border-gray-700">
                    <div className="p-8">
                        <div className="flex items-center mb-6">
                            <div className="w-12 h-12 bg-gradient-to-br from-orange-500 to-red-600 rounded-lg flex items-center justify-center mr-4">
                                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                                </svg>
                            </div>
                            <h3 className="text-xl font-bold text-gray-900 dark:text-white">Enterprise Security</h3>
                        </div>
                        <p className="text-gray-600 dark:text-gray-300 mb-4">
                            Built-in authentication, authorization, and database security features ensure
                            your applications are secure by default.
                        </p>
                        <div className="flex items-center text-sm text-orange-600 dark:text-orange-400 font-medium">
                            <span>Review security features</span>
                            <svg className="ml-1 w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                            </svg>
                        </div>
                    </div>
                </div>
            </div>

            {/* Server Data Section */}
            {data && Object.keys(data).length > 0 && (
                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 overflow-hidden">
                    <div className="px-6 py-4 bg-gray-50 dark:bg-gray-700 border-b border-gray-100 dark:border-gray-600">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                            <svg className="w-5 h-5 text-blue-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                            </svg>
                            Server-Side Data
                        </h3>
                    </div>
                    <div className="p-6">
                        <pre className="text-sm text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-900 p-4 rounded-lg border dark:border-gray-600 overflow-x-auto font-mono">
                            {JSON.stringify(data, null, 2)}
                        </pre>
                    </div>
                </div>
            )}
        </div>
    );
}
