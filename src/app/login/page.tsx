'use client';

import React from 'react';
import { LoginForm, AuthContainer } from '@/lib/codex-auth';
import { useRouter } from 'next/navigation';
import Image from 'next/image';

export default function LoginPage() {
  const router = useRouter();

  const handleSuccess = () => {
    // Redirect to dashboard after successful login
    router.push('/dashboard');
  };

  const handleError = (error: string) => {
    console.error('Login error:', error);
  };

  return (
    <AuthContainer
      title="Welcome to CodexHash"
      subtitle="Sign in to access your quantum-resistant hashing dashboard"
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
      <LoginForm
        onSuccess={handleSuccess}
        onError={handleError}
        showRegisterLink={true}
        registerUrl="/register"
        showForgotPassword={true}
        forgotPasswordUrl="/forgot-password"
      />
    </AuthContainer>
  );
}
