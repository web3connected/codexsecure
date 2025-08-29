#!/usr/bin/env node
/**
 * CodexHash Bridge Script
 * Allows Python benchmark harness to call CodexHash implementation
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {
    const args = process.argv.slice(2);
    
    if (args.length !== 1) {
        console.error('Usage: node codex_hash_bridge.js <hex_input>');
        process.exit(1);
    }
    
    const hexInput = args[0];
    
    try {
        // Try to import CodexHash
        const codexHashPath = path.join(__dirname, '../../../NPMPackages/codexhash/dist/index.esm.js');
        
        if (!fs.existsSync(codexHashPath)) {
            console.error('CodexHash dist not found. Run: cd ../NPMPackages/codexhash && npm run build');
            process.exit(1);
        }
        
        const CodexHash = await import(codexHashPath);
        
        // Convert hex to buffer
        const inputBuffer = Buffer.from(hexInput, 'hex');
        
        // Create CodexHash instance with deterministic configuration
        const config = {
            temporalDynamics: false,  // Disable for deterministic testing
            sacredMatrix: 12,
            inversionLaw: 'dual-polarity',
            securityLevel: 'quantum',
            algorithm: 'universal-law-hash'
        };
        const hasher = new CodexHash.HarmonicHash(config);
        
        // Hash the input using the generate method
        const result = hasher.generate(inputBuffer);
        
        // Output the hash as hex
        console.log(result);
        
    } catch (error) {
        console.error('CodexHash error:', error.message);
        process.exit(1);
    }
}

main().catch(error => {
    console.error('Unexpected error:', error);
    process.exit(1);
});
