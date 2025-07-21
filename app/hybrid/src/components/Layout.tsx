import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { HybridLink } from './HybridLink';

interface LayoutProps {
    children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
    const location = useLocation();
    const currentPath = location.pathname;
    const { isDark } = useTheme();

    const navigation = [
        {
            name: 'Home',
            href: '/hybrid',
            path: '/hybrid',
            icon: 'home',
            dataEndpoint: '/hybrid/data'
        },
        {
            name: 'Settings',
            href: '/hybrid/settings',
            path: '/hybrid/settings',
            icon: 'settings',
            dataEndpoint: '/hybrid/data/settings'
        },
        {
            name: 'About',
            href: '/hybrid/about',
            path: '/hybrid/about',
            icon: 'info',
            dataEndpoint: '/hybrid/data/about'
        },
    ];

    const getIcon = (iconType: string) => {
        switch (iconType) {
            case 'home':
                return (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                    </svg>
                );
            case 'settings':
                return (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                );
            case 'info':
                return (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                );
            default:
                return null;
        }
    };

    return (
        <div className={`min-h-screen transition-colors duration-300 ${isDark ? 'bg-gradient-to-br from-gray-900 to-gray-800' : 'bg-gradient-to-br from-gray-50 to-blue-50'}`}>
            {/* Navigation */}
            <nav className={`${isDark ? 'bg-gray-800/80 border-gray-700' : 'bg-white/80 border-gray-100'} backdrop-blur-md shadow-lg border-b sticky top-0 z-50 transition-colors duration-300`}>
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between h-16">
                        <div className="flex">
                            <div className="flex-shrink-0 flex items-center">
                                <Link to="/" reloadDocument={true} className="flex items-center space-x-2 group">
                                    <div className="flex-shrink-0 flex items-center">
                                        <span className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>Turbo</span>
                                        <span className="text-2xl font-bold text-red-500">Script</span>
                                    </div>
                                </Link>
                            </div>

                            <div className="hidden sm:ml-6 sm:flex sm:space-x-1">
                                {navigation.map((item) => (
                                    <HybridLink
                                        key={item.name}
                                        to={item.href}
                                        dataEndpoint={item.dataEndpoint}
                                        className={`${currentPath === item.path
                                            ? `${isDark ? 'bg-blue-900/50 text-blue-300 border-blue-400' : 'bg-blue-100 text-blue-700 border-blue-500'}`
                                            : `${isDark ? 'text-gray-300 hover:text-blue-400 hover:bg-blue-900/30' : 'text-gray-600 hover:text-blue-600 hover:bg-blue-50'} border-transparent`
                                            } inline-flex items-center px-4 py-2 border-b-2 text-sm font-medium transition-all duration-200 rounded-t-lg group`}
                                    >
                                        <span className={`mr-2 ${currentPath === item.path ? (isDark ? 'text-blue-400' : 'text-blue-600') : (isDark ? 'text-gray-400 group-hover:text-blue-400' : 'text-gray-400 group-hover:text-blue-500')} transition-colors`}>
                                            {getIcon(item.icon)}
                                        </span>
                                        {item.name}
                                    </HybridLink>
                                ))}
                            </div>
                        </div>

                        {/* Mobile menu button */}
                        <div className="sm:hidden flex items-center">
                            <button className={`${isDark ? 'text-gray-300 hover:text-blue-400' : 'text-gray-600 hover:text-blue-600'} focus:outline-none transition-colors`}>
                                <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                                </svg>
                            </button>
                        </div>
                    </div>
                </div>

                {/* Mobile menu */}
                <div className={`sm:hidden border-t ${isDark ? 'border-gray-700 bg-gray-800/95' : 'border-gray-100 bg-white/95'} backdrop-blur-md transition-colors duration-300`}>
                    <div className="pt-2 pb-3 space-y-1">
                        {navigation.map((item) => (
                            <HybridLink
                                key={item.name}
                                to={item.href}
                                dataEndpoint={item.dataEndpoint}
                                className={`${currentPath === item.path
                                    ? `${isDark ? 'bg-blue-900/50 text-blue-300 border-l-4 border-blue-400' : 'bg-blue-100 text-blue-700 border-l-4 border-blue-500'}`
                                    : `${isDark ? 'text-gray-300 hover:text-blue-400 hover:bg-blue-900/30' : 'text-gray-600 hover:text-blue-600 hover:bg-blue-50'} border-l-4 border-transparent`
                                    } block pl-3 pr-4 py-2 text-base font-medium transition-all duration-200`}
                            >
                                <span className="flex items-center">
                                    <span className={`mr-3 ${currentPath === item.path ? (isDark ? 'text-blue-400' : 'text-blue-600') : (isDark ? 'text-gray-400' : 'text-gray-400')}`}>
                                        {getIcon(item.icon)}
                                    </span>
                                    {item.name}
                                </span>
                            </HybridLink>
                        ))}
                    </div>
                </div>
            </nav>

            {/* Main content */}
            <main className="max-w-7xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
                {children}
            </main>

            {/* Footer */}
            <footer className={`${isDark ? 'bg-gray-800/60 border-gray-700' : 'bg-white/60 border-gray-100'} backdrop-blur-md border-t mt-16 transition-colors duration-300`}>
                <div className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between items-center">
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                            © 2025 TurboScript. Powered by TypeScript & Go.
                        </p>
                        <div className="flex space-x-4">
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${isDark ? 'bg-blue-900/50 text-blue-300' : 'bg-blue-100 text-blue-800'}`}>
                                React Router Enabled
                            </span>
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${isDark ? 'bg-green-900/50 text-green-300' : 'bg-green-100 text-green-800'}`}>
                                Server-Side Rendered
                            </span>
                        </div>
                    </div>
                </div>
            </footer>
        </div>
    );
}
