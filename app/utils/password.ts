/**
 * Secure password utilities using Argon2 hashing algorithm
 * Argon2id is the recommended variant providing protection against both side-channel and GPU attacks
 */

/**
 * Default Argon2 configuration optimized for security and performance
 * These values follow OWASP recommendations for password hashing
 */
const ARGON2_CONFIG = {
    type: 2,                  // Argon2id variant (recommended)
    memoryCost: 65536,        // 64 MB memory usage (2^16 KB)
    timeCost: 3,             // 3 iterations
    parallelism: 4,          // 4 parallel threads
    hashLength: 32,          // 32-byte hash output
    saltLength: 16,          // 16-byte salt
};

/**
 * Hash a password using Argon2id algorithm
 * @param password - Plain text password to hash
 * @param options - Optional Argon2 configuration overrides
 * @returns Promise<string> - Argon2 hash string
 */
export const hashPassword = async (password: string, options: Partial<typeof ARGON2_CONFIG> = {}): Promise<string> => {
    if (!password || password.length === 0) {
        throw new Error('Password cannot be empty');
    }

    const config = { ...ARGON2_CONFIG, ...options };

    try {
        const hash = await argon2.hash(password, {
            memoryCost: config.memoryCost,
            timeCost: config.timeCost,
            parallelism: config.parallelism,
            hashLength: config.hashLength,
            saltLength: config.saltLength,
            variant: 'argon2id'
        });
        return hash;
    } catch (error) {
        throw new Error(`Failed to hash password: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
};

/**
 * Verify a password against an Argon2 hash
 * @param password - Plain text password to verify
 * @param hashedPassword - The Argon2 hash from database
 * @returns Promise<boolean> - true if password matches, false otherwise
 */
export const verifyPassword = async (password: string, hashedPassword: string): Promise<boolean> => {
    if (!password || !hashedPassword) {
        return false;
    }

    try {
        // Use argon2.verify with standard (hash, password) parameter order
        return await argon2.verify(hashedPassword, password);
    } catch (error) {
        // Log the error but don't expose details to prevent information leakage
        console.error('Password verification failed:', error instanceof Error ? error.message : 'Unknown error');
        return false;
    }
};

/**
 * Check if a password hash needs to be rehashed due to updated security parameters
 * @param hashedPassword - The current Argon2 hash
 * @param options - New Argon2 configuration to compare against
 * @returns boolean - true if password needs rehashing
 */
export const needsRehash = (hashedPassword: string, options: Partial<typeof ARGON2_CONFIG> = {}): boolean => {
    if (!hashedPassword) {
        return true;
    }

    try {
        const config = { ...ARGON2_CONFIG, ...options };

        // Parse the existing hash to extract current parameters
        // Argon2 hash format: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
        const hashParts = hashedPassword.split('$');

        if (hashParts.length < 6) {
            return true; // Invalid format, needs rehash
        }

        const variant = hashParts[1];
        const params = hashParts[3];

        // Check if variant matches (argon2id is recommended)
        const expectedVariant = 'argon2id';

        if (variant !== expectedVariant) {
            return true;
        }

        // Parse parameters: m=memory,t=time,p=parallelism
        const paramMap = new Map<string, number>();
        for (const param of params.split(',')) {
            const [key, value] = param.split('=');
            paramMap.set(key, parseInt(value, 10));
        }

        const currentMemory = paramMap.get('m') ?? 0;
        const currentTime = paramMap.get('t') ?? 0;
        const currentParallelism = paramMap.get('p') ?? 0;

        // Check if any security parameter has been upgraded
        return (
            currentMemory < config.memoryCost ||
            currentTime < config.timeCost ||
            currentParallelism < config.parallelism
        );
    } catch (_error) {
        return true; // Error parsing, assume needs rehash for safety
    }
};

/**
 * Validate password strength (enhanced security-focused validation)
 * @param password - Password to validate
 * @param options - Validation options
 * @returns validation result with detailed feedback
 */

/**
 * Validate password strength (enhanced security-focused validation)
 * @param password - Password to validate
 * @param options - Validation options
 * @returns validation result with detailed feedback
 */
export const validatePassword = (
    password: string,
    options: {
        minLength?: number;
        maxLength?: number;
        requireUppercase?: boolean;
        requireLowercase?: boolean;
        requireNumbers?: boolean;
        requireSymbols?: boolean;
        forbidCommonPasswords?: boolean;
        forbidPersonalInfo?: string[]; // Array of personal info to check against
    } = {}
): { valid: boolean; errors: string[]; strength: 'weak' | 'fair' | 'good' | 'strong' } => {
    const {
        minLength = 12,              // Increased minimum for better security
        maxLength = 128,             // Reasonable maximum to prevent DoS
        requireUppercase = true,     // More secure defaults
        requireLowercase = true,
        requireNumbers = true,
        requireSymbols = true,
        forbidCommonPasswords = true,
        forbidPersonalInfo = []
    } = options;

    const errors: string[] = [];
    let strengthScore = 0;

    // Length validation
    if (password.length < minLength) {
        errors.push(`Password must be at least ${minLength} characters long`);
    } else {
        strengthScore += Math.min(password.length - minLength, 10); // Bonus points for length
    }

    if (password.length > maxLength) {
        errors.push(`Password must not exceed ${maxLength} characters`);
    }

    // Character requirements
    if (requireUppercase && !/[A-Z]/.test(password)) {
        errors.push('Password must contain at least one uppercase letter');
    } else if (/[A-Z]/.test(password)) {
        strengthScore += 2;
    }

    if (requireLowercase && !/[a-z]/.test(password)) {
        errors.push('Password must contain at least one lowercase letter');
    } else if (/[a-z]/.test(password)) {
        strengthScore += 2;
    }

    if (requireNumbers && !/\d/.test(password)) {
        errors.push('Password must contain at least one number');
    } else if (/\d/.test(password)) {
        strengthScore += 2;
    }

    if (requireSymbols && !/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password)) {
        errors.push('Password must contain at least one symbol (!@#$%^&*()_+-=[]{}|;:,.<>?)');
    } else if (/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password)) {
        strengthScore += 3;
    }

    // Check for character diversity
    const uniqueChars = new Set(password).size;
    if (uniqueChars > password.length * 0.7) {
        strengthScore += 3; // High character diversity
    } else if (uniqueChars > password.length * 0.5) {
        strengthScore += 1; // Moderate diversity
    }

    // Common password patterns to avoid
    if (forbidCommonPasswords) {
        const commonPatterns = [
            /^.{0,2}(.)\1{2,}/, // Repeated characters
            /123456|abcdef|qwerty|password|admin|letmein/i, // Common sequences
            /(.)\1{3,}/, // 4+ consecutive identical characters
        ];

        for (const pattern of commonPatterns) {
            if (pattern.test(password)) {
                errors.push('Password contains common patterns that are easily guessed');
                break;
            }
        }
    }

    // Check against personal information
    if (forbidPersonalInfo.length > 0) {
        const lowerPassword = password.toLowerCase();
        for (const info of forbidPersonalInfo) {
            if (info && info.length > 2 && lowerPassword.includes(info.toLowerCase())) {
                errors.push('Password must not contain personal information');
                break;
            }
        }
    }

    // Determine strength based on score
    let strength: 'weak' | 'fair' | 'good' | 'strong';
    if (strengthScore < 5) {
        strength = 'weak';
    } else if (strengthScore < 10) {
        strength = 'fair';
    } else if (strengthScore < 15) {
        strength = 'good';
    } else {
        strength = 'strong';
    }

    return {
        valid: errors.length === 0,
        errors,
        strength
    };
};

/**
 * Generate a cryptographically secure random password
 * @param length - Length of the password (minimum 12)
 * @param options - Character set options
 * @returns string - Generated password
 */
export const generateSecurePassword = (
    length: number = 16,
    options: {
        includeUppercase?: boolean;
        includeLowercase?: boolean;
        includeNumbers?: boolean;
        includeSymbols?: boolean;
        excludeSimilar?: boolean; // Exclude similar looking characters (0, O, l, 1, etc.)
    } = {}
): string => {
    const {
        includeUppercase = true,
        includeLowercase = true,
        includeNumbers = true,
        includeSymbols = true,
        excludeSimilar = true
    } = options;

    if (length < 12) {
        throw new Error('Password length must be at least 12 characters for security');
    }

    let charset = '';

    if (includeUppercase) {
        charset += excludeSimilar ? 'ABCDEFGHJKMNPQRSTUVWXYZ' : 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
    }

    if (includeLowercase) {
        charset += excludeSimilar ? 'abcdefghjkmnpqrstuvwxyz' : 'abcdefghijklmnopqrstuvwxyz';
    }

    if (includeNumbers) {
        charset += excludeSimilar ? '23456789' : '0123456789';
    }

    if (includeSymbols) {
        charset += '!@#$%^&*()_+-=[]{}|;:,.<>?';
    }

    if (charset.length === 0) {
        throw new Error('At least one character set must be enabled');
    }

    // Use crypto.getRandomValues for cryptographically secure randomness
    const array = new Uint32Array(length);
    globalThis.crypto.getRandomValues(array);

    let password = '';
    for (let i = 0; i < length; i++) {
        password += charset[array[i] % charset.length];
    }

    return password;
};

/**
 * Timing-safe password comparison to prevent timing attacks
 * This is a wrapper around the Argon2 verify function which is already timing-safe
 * @param password - Plain text password
 * @param hashedPassword - Argon2 hash
 * @returns Promise<boolean> - Verification result
 */
export const timingSafeVerify = async (password: string, hashedPassword: string): Promise<boolean> =>
    verifyPassword(password, hashedPassword);
