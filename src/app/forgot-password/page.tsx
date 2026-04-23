'use client';

import React from 'react';
import { PasswordResetForm, AuthContainer } from '@/lib/codex-auth';
import { useRouter } from 'next/navigation';
import Image from 'next/image';

export default function ForgotPasswordPage() {
  const router = useRouter();

  const handleSubmit = async (email: string) => {
    // TODO: Implement actual password reset API call
    console.log('Password reset requested for:', email);
    
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    return {
      success: true,
      message: 'Password reset instructions sent'
    };
  };

  const handleSuccess = () => {
    console.log('Password reset email sent successfully');
  };

  const handleError = (error: string) => {
    console.error('Password reset error:', error);
  };

  return (
    <AuthContainer
      title="Reset Your Password"
      subtitle="We'll send you instructions to reset your password"
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
      <PasswordResetForm
        onSubmit={handleSubmit}
        onSuccess={handleSuccess}
        onError={handleError}
        showBackToLogin={true}
        loginUrl="/login"
      />
    </AuthContainer>
  );
}
