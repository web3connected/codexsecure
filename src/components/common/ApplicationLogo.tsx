import Link from 'next/link';
import React from 'react';

interface ApplicationLogoProps {
    logo: keyof typeof LogoBrands;
    showIcon?: boolean;
}


const LogoBrands = {
    Web3Connected: { 
        pre: 'Web3', 
        post: 'Connected', 
        colorClass: 'logo-web3-codex',
        bgClass: 'bg-gradient-brand'
    },
    CodexTime: { 
        pre: 'Codex', 
        post: 'Time', 
        colorClass: 'logo-codex-time',
        bgClass: 'bg-gradient-time'
    },
    CodexPulse: { 
        pre: 'Codex', 
        post: 'Pulse', 
        colorClass: 'logo-codex-time',
        bgClass: 'bg-gradient-pulse'
    },
    CodexHash: { 
        pre: 'Codex', 
        post: 'Hash', 
        colorClass: 'logo-codex-secure',
        bgClass: 'bg-gradient-secure'
    },
    CodexIdentity: { 
        pre: 'Codex', 
        post: 'Identity', 
        colorClass: 'logo-codex-identity',
        bgClass: 'bg-gradient-identity'
    },
    CodexAuth: { 
        pre: 'Codex', 
        post: 'Auth', 
        colorClass: 'logo-codex-identity',
        bgClass: 'bg-gradient-auth'
    },
    CodexSecure: { 
        pre: 'Codex', 
        post: 'Secure', 
        colorClass: 'logo-codex-secure',
        bgClass: 'bg-gradient-secure'
    }
} as const;

const ApplicationLogo: React.FC<ApplicationLogoProps> = ({ logo, showIcon=true }) => {
    const brand = LogoBrands[logo];

    if (!brand) {
        return <div className="text-red-500">Invalid logo: {logo}</div>;
    }

    return (
        <Link href="/" className="flex items-center gap-3">
            {showIcon && (
                <div 
                    className={`w-10 h-10 rounded-lg flex items-center justify-center animate-pulse-glow ${brand.bgClass}`}
                >
                    <span className="text-white font-bold text-xl brand-text">
                        {brand.pre[0] + brand.post[0]}
                    </span>
                </div>
            )}
            <div className="brand-text text-2xl font-bold">
                <span className="text-trinary">{brand.pre}</span>
                <span className={brand.colorClass}>{brand.post}</span>
            </div>
        </Link>
    );
};

export default ApplicationLogo;
