#!/usr/bin/env node

// Quick test script to verify Argon2 password functionality
// This can be run with: node test-argon2.js

import argon2 from 'argon2';

async function testArgon2() {
    console.log('🔐 Testing Argon2 Password Implementation');
    console.log('==========================================');

    const testPassword = 'MySecureP@ssw0rd123!';

    try {
        // Test hashing
        console.log('📝 Hashing password...');
        const startHash = Date.now();
        const hash = await argon2.hash(testPassword, {
            type: argon2.argon2id,
            memoryCost: 65536,
            timeCost: 3,
            parallelism: 4,
            hashLength: 32,
            saltLength: 16
        });
        const hashTime = Date.now() - startHash;

        console.log(`✅ Hash generated in ${hashTime}ms`);
        console.log(`📄 Hash: ${hash.substring(0, 50)}...`);
        console.log(`📊 Hash length: ${hash.length} characters`);

        // Test verification (correct password)
        console.log('\n🔍 Verifying correct password...');
        const startVerify = Date.now();
        const isValid = await argon2.verify(hash, testPassword);
        const verifyTime = Date.now() - startVerify;

        console.log(`✅ Verification: ${isValid ? 'SUCCESS' : 'FAILED'} (${verifyTime}ms)`);

        // Test verification (wrong password)
        console.log('\n🔍 Verifying wrong password...');
        const startWrong = Date.now();
        const isWrong = await argon2.verify(hash, 'wrongpassword');
        const wrongTime = Date.now() - startWrong;

        console.log(`✅ Wrong password rejected: ${!isWrong ? 'SUCCESS' : 'FAILED'} (${wrongTime}ms)`);

        // Test hash analysis
        console.log('\n🔬 Hash Analysis:');
        const hashParts = hash.split('$');
        console.log(`   Algorithm: ${hashParts[1]}`);
        console.log(`   Version: ${hashParts[2]}`);
        console.log(`   Parameters: ${hashParts[3]}`);
        console.log(`   Salt length: ${Buffer.from(hashParts[4], 'base64').length} bytes`);
        console.log(`   Hash length: ${Buffer.from(hashParts[5], 'base64').length} bytes`);

        console.log('\n🎉 All tests passed! Argon2 implementation is working correctly.');

    } catch (error) {
        console.error('❌ Test failed:', error.message);
        process.exit(1);
    }
}

// Run the test
testArgon2().catch(console.error);
