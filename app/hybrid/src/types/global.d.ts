declare global {
    interface Window {
        __ROUTE_DATA__?: {
            route: string;
            data: unknown;
        };
    }
}

export { };
