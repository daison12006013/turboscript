/**
 * Argon2 password hashing library for goja JavaScript VM
 * Compatible with Node.js argon2 library hash format
 */

declare global {
    /**
     * Argon2 configuration options
     */
    interface Argon2Options {
        /**
         * Memory usage in KB
         * @default 65536 (64MB)
         */
        memoryCost?: number;

        /**
         * Number of iterations
         * @default 3
         */
        timeCost?: number;

        /**
         * Degree of parallelism (number of threads)
         * @default 4
         */
        parallelism?: number;

        /**
         * Length of hash output in bytes
         * @default 32
         */
        hashLength?: number;

        /**
         * Length of salt in bytes
         * @default 16
         */
        saltLength?: number;

        /**
         * Argon2 variant
         * @default 'argon2id'
         */
        variant?: 'argon2id' | 'argon2i' | 'argon2d';
    }

    /**
     * Argon2 password hashing module
     */
    const argon2: {
        /**
         * Asynchronously hash a password using Argon2
         * @param password The password to hash
         * @param options Hashing options
         * @returns Promise that resolves to the encoded hash string
         */
        hash(password: string, options?: Argon2Options): Promise<string>;

        /**
         * Asynchronously verify a password against a hash
         * @param hash The encoded hash string
         * @param password The password to verify
         * @returns Promise that resolves to true if password matches
         */
        verify(hash: string, password: string): Promise<boolean>;

        /**
         * Synchronously hash a password using Argon2
         * @param password The password to hash
         * @param options Hashing options
         * @returns The encoded hash string
         */
        hashSync(password: string, options?: Argon2Options): string;

        /**
         * Synchronously verify a password against a hash
         * @param hash The encoded hash string
         * @param password The password to verify
         * @returns True if password matches
         */
        verifySync(hash: string, password: string): boolean;

        /**
         * Default Argon2 options following OWASP recommendations
         */
        readonly defaults: Required<Argon2Options>;
    };
}

// This export is required to make this file a module
export { };
