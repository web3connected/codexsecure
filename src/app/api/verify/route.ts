import { NextRequest, NextResponse } from 'next/server'
import crypto from 'crypto'

interface VerifyRequest {
  input: string
  hash: string
  salt?: string
  tiu?: number
  iterations?: number
}

interface VerifyResponse {
  valid: boolean
  durationMs: number
  meta: {
    algorithm: string
    inputSize: number
  }
}

// Verify hash by calling the web3connected CodexHash API
async function verifyCodexHash(input: string, hash: string, salt?: string, tiu: number = 0.618034, iterations: number = 16) {
  const backendUrl = 'https://codexhash.web3connected.com'
  
  try {
    const response = await fetch(`${backendUrl}/verify`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'User-Agent': 'CodexHash-Client/1.0.0'
      },
      body: JSON.stringify({
        data: input,
        hash: hash,
        salt: salt,
        tiu: tiu,
        iterations: iterations
      })
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      console.error(`Web3Connected Verify API Error ${response.status}:`, errorText)
      // Fallback to local verification
      return verifyLocalHash(input, hash, salt, tiu, iterations)
    }
    
    return await response.json()
  } catch (error) {
    console.error('Error calling web3connected verify API:', error)
    // Fallback to local verification
    return verifyLocalHash(input, hash, salt, tiu, iterations)
  }
}

// Fallback local verification
function verifyLocalHash(input: string, hash: string, salt?: string, tiu: number = 0.618034, iterations: number = 16) {
  if (!salt) {
    return { valid: false, message: 'Salt required for verification' }
  }
  
  // Recreate hash using same logic as generation
  let testHash = input + salt
  for (let i = 0; i < iterations; i++) {
    testHash = crypto.createHash('sha256').update(testHash + tiu.toString()).digest('hex')
  }
  
  return {
    valid: testHash === hash,
    message: testHash === hash ? 'Hash verified successfully' : 'Hash verification failed'
  }
}

export async function POST(request: NextRequest) {
  const startTime = Date.now()
  
  try {
    const body: VerifyRequest = await request.json()
    
    // Validate required fields
    if (!body.input || !body.hash) {
      return NextResponse.json(
        { error: 'Missing required fields: input and hash are required' },
        { status: 400 }
      )
    }
    
    // Set default values
    const iterations = Math.min(Math.max(body.iterations || 16, 1), 32)
    const tiu = body.tiu || 0.618034 // Golden ratio by default
    const inputSize = new TextEncoder().encode(body.input).length
    
    // Verify hash using backend
    const verifyResult = await verifyCodexHash(body.input, body.hash, body.salt, tiu, iterations)
    
    const response: VerifyResponse = {
      valid: verifyResult.valid || false,
      durationMs: Date.now() - startTime,
      meta: {
        algorithm: 'HarmonicHash-v1.0',
        inputSize
      }
    }
    
    return NextResponse.json(response)
    
  } catch (error) {
    console.error('Hash verification error:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function GET() {
  return NextResponse.json({
    service: 'HarmonicHash Verification API',
    version: '1.0.0',
    description: 'Verify quantum-resistant harmonic hashes',
    endpoints: {
      'POST /api/verify': 'Verify hash against input data',
      'GET /api/verify': 'Get service information'
    },
    parameters: {
      input: 'Original string to verify (required)',
      hash: 'Hash to verify against (required)',
      salt: 'Optional salt value used in original hash',
      tiu: 'Time distortion units (default: 0.618034 - golden ratio)',
      iterations: 'Hash iterations (1-32, default: 16)'
    }
  })
}