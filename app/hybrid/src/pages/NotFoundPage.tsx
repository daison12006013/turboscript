import React from 'react';
import { Link } from 'react-router-dom';

export default function NotFoundPage() {
    return (
        <div className="min-h-[60vh] flex items-center justify-center">
            <div className="text-center max-w-lg mx-auto">
                {/* 404 Illustration */}
                <div className="mb-8">
                    <div className="relative">
                        <div className="text-9xl font-bold text-gray-200 select-none">404</div>
                        <div className="absolute inset-0 flex items-center justify-center">
                            <div className="w-24 h-24 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center">
                                <svg className="w-12 h-12 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                </svg>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Error Message */}
                <h1 className="text-3xl font-bold text-gray-900 mb-4">Page Not Found</h1>
                <p className="text-lg text-gray-600 mb-8 leading-relaxed">
                    Oops! The page you're looking for seems to have wandered off into the digital void.
                    Don't worry, we'll help you find your way back.
                </p>

                {/* Action Buttons */}
                <div className="flex flex-col sm:flex-row gap-4 justify-center">
                    <Link
                        to="/hybrid"
                        className="inline-flex items-center justify-center px-6 py-3 border border-transparent text-base font-medium rounded-lg shadow-sm text-white bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all duration-200"
                    >
                        <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                        </svg>
                        Go Home
                    </Link>

                    <Link
                        to="/hybridabout"
                        className="inline-flex items-center justify-center px-6 py-3 border border-gray-300 text-base font-medium rounded-lg text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all duration-200"
                    >
                        Learn About TurboScript
                    </Link>
                </div>

                {/* Helpful Links */}
                <div className="mt-12 pt-8 border-t border-gray-200">
                    <p className="text-sm text-gray-500 mb-4">Or try one of these popular pages:</p>
                    <div className="flex justify-center space-x-6">
                        <Link to="/hybrid" className="text-blue-600 hover:text-blue-700 text-sm font-medium">
                            Home
                        </Link>
                        <Link to="/hybrid/settings" className="text-blue-600 hover:text-blue-700 text-sm font-medium">
                            Settings
                        </Link>
                        <Link to="/hybrid/about" className="text-blue-600 hover:text-blue-700 text-sm font-medium">
                            About
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
}
