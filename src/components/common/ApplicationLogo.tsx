import Link from 'next/link';
import React from 'react';

/* ===== Brand Catalog ===== */
const LogoBrands = {
    Web3Connected: {
        pre: 'Web3',
        post: 'Connected',
        colorClass: 'logo-web3-connected',
        bgClass: 'bg-gradient-brand',
    },
    CodexTime: {
        pre: 'Codex',
        post: 'Time',
        colorClass: 'logo-codex-time',
        bgClass: 'bg-gradient-time',
    },
    CodexPulse: {
        pre: 'Codex',
        post: 'Pulse',
        colorClass: 'logo-codex-pulse',
        bgClass: 'bg-gradient-pulse',
    },
    CodexHash: {
        pre: 'Codex',
        post: 'Hash',
        colorClass: 'logo-codex-hash',
        bgClass: 'bg-gradient-hash',
    },
    CodexIdentity: {
        pre: 'Codex',
        post: 'Identity',
        colorClass: 'logo-codex-identity',
        bgClass: 'bg-gradient-identity',
    },
    CodexAuth: {
        pre: 'Codex',
        post: 'Auth',
        colorClass: 'logo-codex-auth',
        bgClass: 'bg-gradient-auth',
    },
    CodexSecure: {
        pre: 'Codex',
        post: 'Secure',
        colorClass: 'logo-codex-secure',
        bgClass: 'bg-gradient-secure',
    },
} as const;

type BrandKey = keyof typeof LogoBrands;

/* ===== Sizes ===== */
const sizeClasses = {
    xs: { icon: 'w-6 h-6', iconText: 'text-base', text: 'text-lg' },
    sm: { icon: 'w-8 h-8', iconText: 'text-lg', text: 'text-xl' },
    md: { icon: 'w-10 h-10', iconText: 'text-xl', text: 'text-2xl' },
    lg: { icon: 'w-12 h-12', iconText: 'text-2xl', text: 'text-3xl' },
    xl: { icon: 'w-14 h-14', iconText: 'text-3xl', text: 'text-4xl' },
    '2xl': { icon: 'w-16 h-16', iconText: 'text-4xl', text: 'text-5xl' },
    '3xl': { icon: 'w-20 h-20', iconText: 'text-5xl', text: 'text-5xl lg:text-7xl' },
    '4xl': { icon: 'w-24 h-24', iconText: 'text-6xl', text: 'text-6xl lg:text-8xl' },
    '5xl': { icon: 'w-28 h-28', iconText: 'text-7xl', text: 'text-7xl lg:text-9xl' },
} as const;

type SizeKey = keyof typeof sizeClasses;

/* ===== Props ===== */
interface ApplicationLogoProps {
    logo: BrandKey;
    showIcon?: boolean;
    size?: SizeKey;
    href?: string;
    className?: string;
    center?: boolean; // NEW: allow optional centering
}

/* ===== Component ===== */
const ApplicationLogo: React.FC<ApplicationLogoProps> = ({
    logo,
    showIcon = true,
    size = 'md',
    href = '/',
    className = '',
    center = false,
}) => {
    const brand = LogoBrands[logo];
    const sz = sizeClasses[size] ?? sizeClasses.md;

    const first = (s: string) => (s.match(/[A-Za-z0-9]/)?.[0] ?? '').toUpperCase();
    const initials = `${first(brand.pre)}${first(brand.post)}`;

    const logoElement = (
        <Link
            href={href}
            aria-label={`${brand.pre} ${brand.post}`}
            className={`inline-flex items-center gap-3 w-auto ${className}`} // FIXED: inline-flex + no full width
        >
            {showIcon && (
                <div
                    className={`rounded-lg flex items-center justify-center animate-pulse-glow ${brand.bgClass} ${sz.icon}`}
                >
                    <span className={`text-white font-bold ${sz.iconText}`}>{initials}</span>
                </div>
            )}
            <div className={`font-bold ${sz.text}`}>
                <span className="text-trinary">{brand.pre}</span>
                <span className={`ml-1 ${brand.colorClass}`}>{brand.post}</span>
            </div>
        </Link>
    );

    // If center is true, wrap in a flex container
    return center ? <div className="flex justify-center">{logoElement}</div> : logoElement;
};

export default ApplicationLogo;
