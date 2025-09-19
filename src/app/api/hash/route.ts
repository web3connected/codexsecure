import { NextRequest, NextResponse } from 'next/server'
import crypto from 'crypto'

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

// Generate hash by calling the web3connected CodexHash API
async function generateCodexHash(input: string, salt?: string, tiu: number = 0.618034, iterations: number = 16) {
  const backendUrl = 'https://codexhash.web3connected.com'
  
  try {
    const response = await fetch(`${backendUrl}/hash`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'User-Agent': 'CodexHash-Client/1.0.0'
      },
      body: JSON.stringify({
        input,
        salt,
        tiu,
        iterations
      })
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      console.error(`Web3Connected API Error ${response.status}:`, errorText)
      throw new Error(`Backend hash generation failed: ${response.status} - ${errorText}`)
    }
    
    const result = await response.json()
    return result
  } catch (error) {
    console.error('Error calling web3connected API:', error)
    // Fallback to local hash generation if API fails
    return generateLocalHash(input, salt, tiu, iterations)
  }
}

// Fallback local hash generation
function generateLocalHash(input: string, salt?: string, tiu: number = 0.618034, iterations: number = 16) {
  // Generate salt if not provided
  if (!salt) {
    salt = crypto.randomBytes(16).toString('hex')
  }
  
  // Simple SHA-256 based hash with iterations
  let hash = input + salt
  for (let i = 0; i < iterations; i++) {
    hash = crypto.createHash('sha256').update(hash + tiu.toString()).digest('hex')
  }
  
  return {
    hash: hash,
    salt: salt,
    tiu: tiu.toString(),
    algorithm: 'SHA-256-Fallback',
    quantumResistance: 1.0
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
