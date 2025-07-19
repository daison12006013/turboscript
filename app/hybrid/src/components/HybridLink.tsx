import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useHybridNavigation } from '../hooks/useHybridNavigation';

interface HybridLinkProps {
    to: string;
    dataEndpoint: string;
    children: React.ReactNode;
    className?: string;
    onClick?: () => void;
}

/**
 * HYBRID-aware Link component that fetches data before navigation
 * This ensures that data is loaded from your TurboScript endpoints during navigation
 */
export function HybridLink({ to, dataEndpoint, children, className, onClick }: HybridLinkProps) {
    const navigate = useNavigate();
    const { navigateWithData, isNavigating } = useHybridNavigation();

    const handleClick = async (e: React.MouseEvent<HTMLAnchorElement>) => {
        e.preventDefault();

        // Call custom onClick if provided
        if (onClick) {
            onClick();
        }

        // Navigate with data fetching
        await navigateWithData(to, dataEndpoint, navigate);
    };

    return (
        <Link
            to={to}
            className={`${className || ''} ${isNavigating ? 'opacity-75 pointer-events-none' : ''}`}
            onClick={handleClick}
        >
            {isNavigating ? (
                <span className="flex items-center">
                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-current" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Loading...
                </span>
            ) : (
                children
            )}
        </Link>
    );
}
