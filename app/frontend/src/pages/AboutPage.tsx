import React from 'react';
import { Link } from 'react-router-dom';
import { useHybridDataWithNavigation } from '../hooks/useHybridNavigation';
import { HybridLink } from '../components/HybridLink';

interface AboutData {
    message?: string;
    framework?: string;
    version?: string;
    description?: string;
    features?: string[];
    architecture?: {
        frontend?: string;
        backend?: string;
        runtime?: string;
        database?: string;
        cache?: string;
    };
    performance?: {
        uptime?: string;
        avgResponseTime?: string;
        concurrentUsers?: string;
        memoryFootprint?: string;
    };
    timestamp?: string;
    error?: string;
}

export default function AboutPage() {
    // Use HYBRID data hook to fetch data from /frontend/data/about endpoint
    const { data, loading, error } = useHybridDataWithNavigation<AboutData>(
        '/frontend/data/about',
        '/frontend/about'
    );

    const features = [
        {
            title: 'Hybrid Architecture',
            description: 'TypeScript for business logic, Go for runtime execution',
            icon: '⚡',
            color: 'from-yellow-400 to-orange-500'
        },
        {
            title: 'Real-time Execution',
            description: 'JavaScript VM (goja) for dynamic TypeScript execution',
            icon: '🚀',
            color: 'from-blue-400 to-blue-600'
        },
        {
            title: 'Built-in Security',
            description: 'Authentication, authorization, and database security',
            icon: '🔒',
            color: 'from-green-400 to-green-600'
        },
        {
            title: 'Advanced Caching',
            description: 'Multiple cache drivers for optimal performance',
            icon: '⚡',
            color: 'from-purple-400 to-purple-600'
        },
        {
            title: 'Background Jobs',
            description: 'Asynchronous task processing and scheduling',
            icon: '⏰',
            color: 'from-pink-400 to-pink-600'
        },
        {
            title: 'Email Integration',
            description: 'Multiple email providers with templating support',
            icon: '📧',
            color: 'from-indigo-400 to-indigo-600'
        }
    ];

    const techStack = [
        { name: 'TypeScript', icon: '🔷', description: 'Type-safe business logic' },
        { name: 'Go', icon: '🔵', description: 'High-performance runtime' },
        { name: 'React', icon: '⚛️', description: 'Modern frontend framework' },
        { name: 'FastHTTP', icon: '🌐', description: 'Ultra-fast web server' },
        { name: 'Goja', icon: '⚙️', description: 'JavaScript VM in Go' },
        { name: 'ESBuild', icon: '📦', description: 'Lightning-fast bundler' }
    ];

    const stats = [
        { label: 'Framework Version', value: data?.version || '1.0.0', icon: '🏷️' },
        { label: 'Runtime Performance', value: '< 1ms', icon: '⚡' },
        { label: 'Memory Efficiency', value: '99.2%', icon: '💾' },
        { label: 'Type Safety', value: '100%', icon: '✅' }
    ];

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
                        Loading about information...
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
                            <h3 className="text-sm font-medium text-red-800">Error loading about information</h3>
                            <div className="mt-2 text-sm text-red-700">
                                <p>{error}</p>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Hero Section */}
            <div className="relative bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 rounded-2xl overflow-hidden">
                <div className="absolute inset-0 bg-gradient-to-r from-purple-800/20 to-pink-800/20"></div>
                <div className="relative px-8 py-12 sm:px-12 sm:py-16">
                    <div className="text-center">
                        <div className="inline-flex items-center justify-center w-20 h-20 bg-gradient-to-br from-blue-400 to-purple-600 rounded-full mb-6">
                            <span className="text-3xl font-bold text-white">T</span>
                        </div>
                        <h1 className="text-4xl sm:text-5xl font-bold text-white mb-6">
                            About{' '}
                            <span className="bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
                                TurboScript
                            </span>
                        </h1>
                        <p className="text-xl text-slate-300 max-w-3xl mx-auto leading-relaxed">
                            A revolutionary hybrid web framework that bridges TypeScript's developer experience
                            with Go's performance excellence.
                        </p>
                    </div>
                </div>

                {/* Floating decorative elements */}
                <div className="absolute top-8 left-8 w-12 h-12 bg-blue-400/20 rounded-full animate-pulse"></div>
                <div className="absolute bottom-8 right-8 w-16 h-16 bg-purple-400/20 rounded-full animate-pulse"></div>
                <div className="absolute top-1/2 right-16 w-8 h-8 bg-pink-400/30 rounded-full animate-bounce"></div>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
                {stats.map((stat, index) => (
                    <div key={index} className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 p-6 text-center hover:shadow-xl transition-shadow">
                        <div className="text-3xl mb-2">{stat.icon}</div>
                        <div className="text-2xl font-bold text-gray-900 dark:text-white mb-1">{stat.value}</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">{stat.label}</div>
                    </div>
                ))}
            </div>

            {/* Architecture Overview */}
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 overflow-hidden">
                <div className="bg-gradient-to-r from-blue-500 to-purple-600 px-8 py-6">
                    <h2 className="text-2xl font-bold text-white flex items-center">
                        <svg className="w-8 h-8 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                        </svg>
                        Hybrid Architecture
                    </h2>
                </div>
                <div className="p-8">
                    <div className="prose max-w-none">
                        <p className="text-gray-600 dark:text-gray-300 text-lg leading-relaxed mb-6">
                            TurboScript revolutionizes web development by combining TypeScript's type safety and
                            developer experience with Go's runtime performance. This unique architecture enables
                            you to write business logic in TypeScript while leveraging Go's speed for execution.
                        </p>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mt-8">
                            <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-6">
                                <h3 className="text-lg font-semibold text-blue-900 dark:text-blue-300 mb-3 flex items-center">
                                    <span className="text-2xl mr-2">🔷</span>
                                    TypeScript Layer
                                </h3>
                                <ul className="text-blue-800 dark:text-blue-300 space-y-2">
                                    <li>• Business logic and API handlers</li>
                                    <li>• Type-safe database queries</li>
                                    <li>• Authentication and validation</li>
                                    <li>• Async/await pattern support</li>
                                </ul>
                            </div>

                            <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-6">
                                <h3 className="text-lg font-semibold text-green-900 dark:text-green-300 mb-3 flex items-center">
                                    <span className="text-2xl mr-2">🔵</span>
                                    Go Runtime
                                </h3>
                                <ul className="text-green-800 dark:text-green-300 space-y-2">
                                    <li>• FastHTTP web server</li>
                                    <li>• JavaScript VM (goja) execution</li>
                                    <li>• Database connection pooling</li>
                                    <li>• Memory-efficient processing</li>
                                </ul>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Features Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {features.map((feature, index) => (
                    <div key={index} className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 overflow-hidden hover:shadow-xl transition-shadow">
                        <div className={`h-2 bg-gradient-to-r ${feature.color}`}></div>
                        <div className="p-6">
                            <div className="flex items-center mb-4">
                                <div className="text-3xl mr-3">{feature.icon}</div>
                                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{feature.title}</h3>
                            </div>
                            <p className="text-gray-600 dark:text-gray-300">{feature.description}</p>
                        </div>
                    </div>
                ))}
            </div>

            {/* Technology Stack */}
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700">
                <div className="px-8 py-6 border-b border-gray-100 dark:border-gray-700">
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center">
                        <svg className="w-8 h-8 mr-3 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
                        </svg>
                        Technology Stack
                    </h2>
                </div>
                <div className="p-8">
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-6">
                        {techStack.map((tech, index) => (
                            <div key={index} className="text-center p-4 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors">
                                <div className="text-4xl mb-2">{tech.icon}</div>
                                <h3 className="font-semibold text-gray-900 dark:text-white mb-1">{tech.name}</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-300">{tech.description}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* Getting Started */}
            <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl overflow-hidden">
                <div className="px-8 py-12 text-center">
                    <h2 className="text-3xl font-bold text-white mb-4">Ready to Get Started?</h2>
                    <p className="text-blue-100 text-lg mb-8 max-w-2xl mx-auto">
                        Experience the power of TurboScript by exploring our settings page or diving into the documentation.
                    </p>
                    <div className="flex flex-col sm:flex-row justify-center gap-4">
                        <HybridLink
                            to="/frontend/settings"
                            dataEndpoint="/frontend/data/settings"
                            className="inline-flex items-center justify-center px-6 py-3 bg-white text-blue-600 font-semibold rounded-lg hover:bg-blue-50 transition-all duration-200 shadow-lg hover:shadow-xl"
                        >
                            Explore Settings
                            <svg className="ml-2 w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            </svg>
                        </HybridLink>
                        <HybridLink
                            to="/frontend"
                            dataEndpoint="/frontend/data"
                            className="inline-flex items-center justify-center px-6 py-3 bg-blue-500/20 text-white font-semibold rounded-lg hover:bg-blue-500/30 transition-all duration-200 border border-white/20 backdrop-blur-sm"
                        >
                            Back to Home
                        </HybridLink>
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
