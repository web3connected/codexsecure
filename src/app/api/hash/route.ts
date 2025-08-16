import { NextRequest, NextResponse } from 'next/server'
import { HarmonicHash, QuantumHasher } from '@web3connected/codexhash'

interface HashRequest {
  input: string
  salt?: string
  tiu?: number
  iterations?: number
}

interface HashResponse {
  hash: string
  salt: string
  tiu: number
  meta: {
    algo: string
    iterations: number
    durationMs: number
    inputSize: number
    quantumResistance: number
  }
}

// Generate hash using our HarmonicHash implementation
async function generateCodexHash(input: string, salt?: string, tiu: number = 0.618034, iterations: number = 16) {
  const harmonicHash = new HarmonicHash()
  const quantumHasher = new QuantumHasher()
  
  // Generate harmonic hash
  const result = harmonicHash.hash(input, salt, tiu, iterations)
  
  // Calculate quantum resistance
  const quantumResistance = quantumHasher.calculateQuantumResistance(result.hash)
  
  return {
    ...result,
    quantumResistance
  }
}

export async function POST(request: NextRequest) {
  const startTime = Date.now()
  
  try {
    const body: HashRequest = await request.json()
    
    if (!body.input || typeof body.input !== 'string') {
      return NextResponse.json(
        { error: 'Input is required and must be a string' },
        { status: 400 }
      )
    }
    
    // Input size limit (1MB)
    if (body.input.length > 1024 * 1024) {
      return NextResponse.json(
        { error: 'Input too large. Maximum size is 1MB' },
        { status: 413 }
      )
    }
    
    const iterations = Math.min(Math.max(body.iterations || 16, 1), 32) // Clamp between 1-32
    const tiu = body.tiu || 0.618034 // Golden ratio by default
    const inputSize = new TextEncoder().encode(body.input).length
    
    // Generate hash using HarmonicHash
    const hashResult = await generateCodexHash(body.input, body.salt, tiu, iterations)
    
    const response: HashResponse = {
      hash: hashResult.hash,
      salt: hashResult.salt,
      tiu: parseFloat(hashResult.tiu),
      meta: {
        algo: 'HarmonicHash-v1.0',
        iterations,
        durationMs: Date.now() - startTime,
        inputSize,
        quantumResistance: hashResult.quantumResistance
      }
    }
    
    return NextResponse.json(response)
    
  } catch (error) {
    console.error('Hash generation error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function GET() {
  return NextResponse.json({
    service: 'HarmonicHash API',
    version: '1.0.0',
    description: 'Quantum-resistant hashing using physics-based constants and harmonic frequencies',
    endpoints: {
      'POST /api/hash': 'Generate harmonic hash',
      'GET /api/hash': 'Get service information'
    },
    parameters: {
      input: 'String to hash (required)',
      salt: 'Optional salt value',
      tiu: 'Time distortion units (default: 0.618034 - golden ratio)',
      iterations: 'Hash iterations (1-32, default: 16)'
    }
  })
}
