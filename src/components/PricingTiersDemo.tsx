'use client';

import React, { useState, useEffect, useCallback } from 'react';

// Pricing tier interface
interface PriceTier {
  id: string;
  name: string;
  basePrice: number;
  features: string[];
  quantumResistance: number;
  securityLevel: string;
  popular?: boolean;
}

const PRICE_TIERS: PriceTier[] = [
  {
    id: 'starter',
    name: 'Starter',
    basePrice: 9.99,
    features: ['Basic Hashing', 'Standard Security', '10GB Storage'],
    quantumResistance: 0.3,
    securityLevel: 'Standard'
  },
  {
    id: 'professional',
    name: 'Professional',
    basePrice: 29.99,
    features: ['Harmonic Hashing', 'Time-Aware Security', '100GB Storage', 'Enhanced Protection'],
    quantumResistance: 0.7,
    securityLevel: 'Enhanced',
    popular: true
  },
  {
    id: 'quantum',
    name: 'Quantum',
    basePrice: 99.99,
    features: ['Quantum-Resistant', 'Sacred Matrix', '1TB Storage', 'Universal Law Principles'],
    quantumResistance: 0.85,
    securityLevel: 'Quantum'
  },
  {
    id: 'sacred',
    name: 'Sacred',
    basePrice: 299.99,
    features: ['Sacred Geometry', 'Golden Ratio Algorithms', '10TB Storage', 'Fibonacci Sequences'],
    quantumResistance: 0.95,
    securityLevel: 'Sacred'
  }
];

export default function PricingTiersDemo() {
  const [userProfile, setUserProfile] = useState({
    industry: 'technology',
    dataVolume: 100,
    securityNeeds: 'medium',
    region: 'us',
    contractLength: 12
  });
  const [calculatedPricing, setCalculatedPricing] = useState<{
    tiers: (PriceTier & { finalPrice?: number; discount?: number; recommended?: boolean })[];
    multipliers: {
      industry: number;
      volume: number;
      region: number;
      discount: number;
    };
    hash: string;
    salt: string;
    tiu: string;
  } | null>(null);
  const [quantumAnalysis, setQuantumAnalysis] = useState<{
    securityHash: string;
    quantumResistance: number;
    recommendedTier?: PriceTier;
    baseFrequency?: number;
  } | null>(null);

  const getIndustryMultiplier = useCallback((industry: string): number => {
    const multipliers: Record<string, number> = {
      'finance': 1.5,
      'healthcare': 1.4,
      'technology': 1.2,
      'gaming': 1.3,
      'education': 0.9,
      'retail': 1.1,
      'enterprise': 1.6,
      'startup': 0.8
    };
    return multipliers[industry] || 1.0;
  }, []);

  const getRegionMultiplier = useCallback((region: string): number => {
    const multipliers: Record<string, number> = {
      'us': 1.0,
      'eu': 1.1,
      'asia': 0.8,
      'latam': 0.7
    };
    return multipliers[region] || 1.0;
  }, []);

  const getContractDiscount = useCallback((months: number): number => {
    if (months >= 36) return 0.25;
    if (months >= 24) return 0.20;
    if (months >= 12) return 0.15;
    if (months >= 6) return 0.10;
    return 0;
  }, []);

  useEffect(() => {
    const calculateDynamicPricing = async () => {
      try {
        // Create user profile hash for pricing using the local API
        const profileData = {
          ...userProfile,
          timestamp: Math.floor(Date.now() / (1000 * 60 * 60 * 24)) // Daily pricing
        };
        
        const hashResponse = await fetch('/api/hash', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            input: JSON.stringify(profileData),
            tiu: 0.618034, // Golden ratio TIU
            iterations: 16
          })
        });
        
        if (!hashResponse.ok) {
          throw new Error('Hash generation failed');
        }
        
        const profileHash = await hashResponse.json();
        
        // Calculate a simple quantum resistance metric (simplified)
        const quantumResistance = Math.min(0.95, 0.8 + (profileHash.meta.quantumResistance / 100) * 0.15);
        
        // Calculate multipliers
        const industryMultiplier = getIndustryMultiplier(userProfile.industry);
        const volumeMultiplier = 1 + (userProfile.dataVolume / 1000) * 0.1;
        const regionMultiplier = getRegionMultiplier(userProfile.region);
        const contractDiscount = getContractDiscount(userProfile.contractLength);
        
        // Update pricing for each tier
        const updatedTiers = PRICE_TIERS.map(tier => {
          const adjustedPrice = tier.basePrice * industryMultiplier * volumeMultiplier * regionMultiplier;
          const finalPrice = adjustedPrice * (1 - contractDiscount);
          
          return {
            ...tier,
            finalPrice: Math.round(finalPrice * 100) / 100,
            discount: contractDiscount * 100,
            recommended: quantumResistance >= tier.quantumResistance * 0.8
          };
        });
        
        setCalculatedPricing({
          tiers: updatedTiers,
          multipliers: {
            industry: industryMultiplier,
            volume: volumeMultiplier,
            region: regionMultiplier,
            discount: contractDiscount
          },
          hash: profileHash.hash,
          salt: profileHash.salt,
          tiu: profileHash.tiu
        });
        
        setQuantumAnalysis({
          securityHash: profileHash.hash.substring(0, 32), // First 32 chars as security hash
          quantumResistance: Math.round(quantumResistance * 100),
          recommendedTier: updatedTiers.find(t => t.recommended),
          baseFrequency: 432.0 // Default harmonic frequency
        });
        
      } catch (error) {
        console.error('Pricing calculation error:', error);
      }
    };

    calculateDynamicPricing();
  }, [userProfile, getIndustryMultiplier, getRegionMultiplier, getContractDiscount]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-gray-900 py-12 px-4">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-white mb-4">
            Quantum-Resistant Pricing Tiers
          </h1>
          <p className="text-xl text-gray-300 mb-6">
            Dynamic pricing powered by Harmonic Hash algorithms
          </p>
          
          {quantumAnalysis && (
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-lg p-6 max-w-4xl mx-auto mb-8">
              <h3 className="text-lg font-semibold text-white mb-4">Quantum Security Analysis</h3>
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-400">{quantumAnalysis.quantumResistance}%</div>
                  <div className="text-gray-400">Quantum Resistance</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-400">{quantumAnalysis.baseFrequency?.toFixed(10)}</div>
                  <div className="text-gray-400">Base Frequency (Hz)</div>
                </div>
                <div className="text-center">
                  <div className="text-lg font-bold text-purple-400">{quantumAnalysis.recommendedTier?.name}</div>
                  <div className="text-gray-400">Recommended Tier</div>
                </div>
                <div className="text-center">
                  <div className="text-sm font-mono text-yellow-400">{quantumAnalysis.securityHash?.substring(0, 12)}...</div>
                  <div className="text-gray-400">Security Hash</div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* User Profile Configuration */}
        <div className="bg-gray-800/50 backdrop-blur-sm rounded-lg p-6 mb-8">
          <h3 className="text-xl font-semibold text-white mb-4">Customize Your Profile for Dynamic Pricing</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Industry</label>
              <select 
                value={userProfile.industry}
                onChange={(e) => setUserProfile({...userProfile, industry: e.target.value})}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
              >
                <option value="technology">Technology</option>
                <option value="finance">Finance</option>
                <option value="healthcare">Healthcare</option>
                <option value="gaming">Gaming</option>
                <option value="retail">Retail</option>
                <option value="education">Education</option>
              </select>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Data Volume (GB/month)</label>
              <input
                type="number"
                value={userProfile.dataVolume}
                onChange={(e) => setUserProfile({...userProfile, dataVolume: parseInt(e.target.value)})}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Security Needs</label>
              <select 
                value={userProfile.securityNeeds}
                onChange={(e) => setUserProfile({...userProfile, securityNeeds: e.target.value})}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Region</label>
              <select 
                value={userProfile.region}
                onChange={(e) => setUserProfile({...userProfile, region: e.target.value})}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
              >
                <option value="us">United States</option>
                <option value="eu">Europe</option>
                <option value="asia">Asia</option>
                <option value="latam">Latin America</option>
              </select>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Contract (months)</label>
              <select 
                value={userProfile.contractLength}
                onChange={(e) => setUserProfile({...userProfile, contractLength: parseInt(e.target.value)})}
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white"
              >
                <option value="1">Monthly</option>
                <option value="6">6 Months</option>
                <option value="12">12 Months</option>
                <option value="24">24 Months</option>
                <option value="36">36 Months</option>
              </select>
            </div>
          </div>
        </div>

        {/* Pricing Tiers */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
          {(calculatedPricing?.tiers || PRICE_TIERS).map((tier: PriceTier & { finalPrice?: number; discount?: number; recommended?: boolean }) => (
            <div 
              key={tier.id}
              className={`relative bg-gray-800/50 backdrop-blur-sm rounded-lg p-6 border-2 transition-all duration-300 hover:scale-105 cursor-pointer ${
                tier.popular 
                  ? 'border-purple-500 shadow-lg shadow-purple-500/20' 
                  : tier.recommended 
                    ? 'border-green-500 shadow-lg shadow-green-500/20'
                    : 'border-gray-700 hover:border-gray-600'
              }`}
            >
              {tier.popular && (
                <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                  <span className="bg-purple-600 text-white px-4 py-1 rounded-full text-sm font-medium">
                    Most Popular
                  </span>
                </div>
              )}
              
              {tier.recommended && (
                <div className="absolute -top-4 right-4">
                  <span className="bg-green-600 text-white px-3 py-1 rounded-full text-xs font-medium">
                    Recommended
                  </span>
                </div>
              )}

              <div className="text-center mb-6">
                <h3 className="text-2xl font-bold text-white mb-2">{tier.name}</h3>
                <div className="flex items-center justify-center">
                  <span className="text-4xl font-bold text-white">
                    ${tier.finalPrice || tier.basePrice}
                  </span>
                  <span className="text-gray-400 ml-2">/month</span>
                </div>
                
                {(tier.discount || 0) > 0 && (
                  <div className="mt-2">
                    <span className="text-green-400 text-sm">
                      {(tier.discount || 0).toFixed(0)}% discount applied
                    </span>
                  </div>
                )}
                
                <div className="mt-3">
                  <div className="text-sm text-gray-400">Quantum Resistance</div>
                  <div className="w-full bg-gray-700 rounded-full h-2 mt-1">
                    <div 
                      className="bg-gradient-to-r from-blue-500 to-purple-500 h-2 rounded-full"
                      style={{ width: `${tier.quantumResistance * 100}%` }}
                    ></div>
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    {(tier.quantumResistance * 100).toFixed(0)}% resistance
                  </div>
                </div>
              </div>

              <div className="space-y-3 mb-6">
                {tier.features.map((feature: string, index: number) => (
                  <div key={index} className="flex items-center text-gray-300">
                    <svg className="w-4 h-4 text-green-400 mr-3 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    <span className="text-sm">{feature}</span>
                  </div>
                ))}
              </div>

              <button className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${
                tier.popular 
                  ? 'bg-purple-600 hover:bg-purple-700 text-white'
                  : tier.recommended
                    ? 'bg-green-600 hover:bg-green-700 text-white'
                    : 'bg-gray-700 hover:bg-gray-600 text-white'
              }`}>
                Choose {tier.name}
              </button>
              
              <div className="mt-3 text-xs text-gray-500 text-center">
                Security Level: {tier.securityLevel}
              </div>
            </div>
          ))}
        </div>

        {/* Pricing Details */}
        {calculatedPricing && (
          <div className="mt-12 bg-gray-800/50 backdrop-blur-sm rounded-lg p-6">
            <h3 className="text-xl font-semibold text-white mb-4">Pricing Calculation Details</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <h4 className="text-lg font-medium text-white mb-3">Multipliers Applied</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Industry ({userProfile.industry}):</span>
                    <span className="text-white">{calculatedPricing.multipliers.industry}x</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Volume ({userProfile.dataVolume}GB):</span>
                    <span className="text-white">{calculatedPricing.multipliers.volume.toFixed(2)}x</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Region ({userProfile.region}):</span>
                    <span className="text-white">{calculatedPricing.multipliers.region}x</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Contract Discount:</span>
                    <span className="text-green-400">-{(calculatedPricing.multipliers.discount * 100).toFixed(0)}%</span>
                  </div>
                </div>
              </div>
              
              <div>
                <h4 className="text-lg font-medium text-white mb-3">Harmonic Hash Verification</h4>
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="text-gray-400">Pricing Hash:</span>
                    <div className="font-mono text-xs text-blue-400 break-all mt-1">
                      {calculatedPricing.hash.substring(0, 32)}...
                    </div>
                  </div>
                  <div>
                    <span className="text-gray-400">Salt:</span>
                    <div className="font-mono text-xs text-yellow-400 break-all mt-1">
                      {calculatedPricing.salt.substring(0, 16)}...
                    </div>
                  </div>
                  <div>
                    <span className="text-gray-400">TIU (Time Distortion Unit):</span>
                    <div className="font-mono text-xs text-purple-400 mt-1">
                      {calculatedPricing.tiu}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
