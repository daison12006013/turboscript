import crypto from 'node:crypto';

interface SignedUrlOptions {
    route: string;
    params?: Record<string, string | number>;
    expiresInMinutes?: number;
    secret: string;
}

export class SignedUrl {
    static generate({ route, params = {}, expiresInMinutes, secret }: SignedUrlOptions): string {
        // Create URL manually since node:url is not available in Goja
        const searchParams = new URLSearchParams();

        // Add params
        for (const [key, value] of Object.entries(params)) {
            searchParams.append(key, value.toString());
        }

        // Add expiration timestamp if needed
        if (expiresInMinutes) {
            const expires = Math.floor(Date.now() / 1000) + expiresInMinutes * 60;
            searchParams.append("expires", expires.toString());
        }

        const queryString = searchParams.toString();
        const signatureInput = queryString ? `${route}?${queryString}` : route;
        const signature = SignedUrl.sign(signatureInput, secret);
        searchParams.append("signature", signature);

        const finalQuery = searchParams.toString();
        return finalQuery ? `${route}?${finalQuery}` : route;
    }

    static validate(fullUrl: string, secret: string): boolean {
        // Parse URL manually since node:url is not available
        const [baseUrl, queryString] = fullUrl.split('?', 2);
        if (!queryString) return false;

        const searchParams = new URLSearchParams(queryString);
        const signature = searchParams.get("signature");
        if (!signature) return false;

        const expires = searchParams.get("expires");
        if (expires && parseInt(expires) < Math.floor(Date.now() / 1000)) return false;

        // Rebuild query without the signature
        searchParams.delete("signature");
        const rawQuery = searchParams.toString();
        const raw = rawQuery ? `${baseUrl}?${rawQuery}` : baseUrl;
        const expected = SignedUrl.sign(raw, secret);

        return expected === signature; // Use simple comparison instead of timingSafeEqual
    }

    private static sign(value: string, secret: string): string {
        // Use TurboScript's built-in crypto utilities
        return crypto.createHmac('sha256', secret).update(value).digest('hex');
    }
}