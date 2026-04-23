'use client';

import React, { useState } from 'react';
import { AuthContainer, useCodexAuth } from '@/lib/codex-auth';
import { useRouter } from 'next/navigation';
import Image from 'next/image';

export default function RegisterPage() {
  const router = useRouter();
  const { login, loading } = useCodexAuth();
  
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    confirmPassword: '',
  });
  const [error, setError] = useState('');

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({
      ...prev,
      [e.target.name]: e.target.value
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (!formData.name || !formData.email || !formData.password) {
      setError('Please fill in all fields');
      return;
    }

    if (formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (formData.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    try {
      // TODO: Implement actual registration API call
      // For now, just show success and redirect to login
      console.log('Registration data:', formData);
      alert('Registration successful! Please login.');
      router.push('/login');
    } catch (err: any) {
      setError(err.message || 'Registration failed');
    }
  };

  return (
    <AuthContainer
      title="Create Your Account"
      subtitle="Join CodexHash for quantum-resistant security"
      logo={
        <div className="relative w-20 h-20">
          <Image
            src="/shared-template/images/CodexHash.png"
            alt="CodexHash"
            fill
            className="object-contain"
          />
        </div>
      }
      showCodexBranding={true}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
            {error}
          </div>
        )}

        <div className="form-group">
          <label htmlFor="name" className="block text-sm font-medium mb-2">
            Full Name
          </label>
          <input
            id="name"
            name="name"
            type="text"
            value={formData.name}
            onChange={handleChange}
            className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900"
            placeholder="John Doe"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="email" className="block text-sm font-medium mb-2">
            Email Address
          </label>
          <input
            id="email"
            name="email"
            type="email"
            value={formData.email}
            onChange={handleChange}
            className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900"
            placeholder="your@email.com"
            disabled={loading}
            required
          />
        </div>

        <div className="form-group">
          <label htmlFor="password" className="block text-sm font-medium mb-2">
            Password
          </label>
          <input
            id="password"
            name="password"
            type="password"
            value={formData.password}
            onChange={handleChange}
            className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900"
            placeholder="••••••••"
            disabled={loading}
            required
          />
          <p className="text-xs text-gray-500 mt-1">
            Must be at least 8 characters
          </p>
        </div>

        <div className="form-group">
          <label htmlFor="confirmPassword" className="block text-sm font-medium mb-2">
            Confirm Password
          </label>
          <input
            id="confirmPassword"
            name="confirmPassword"
            type="password"
            value={formData.confirmPassword}
            onChange={handleChange}
            className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900"
            placeholder="••••••••"
            disabled={loading}
            required
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? 'Creating Account...' : 'Create Account'}
        </button>

        <div className="text-center text-sm">
          Already have an account?{' '}
          <a
            href="/login"
            className="text-blue-600 hover:text-blue-800 font-medium"
          >
            Sign in here
          </a>
        </div>
      </form>
    </AuthContainer>
  );
}
