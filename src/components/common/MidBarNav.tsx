import React from 'react'

import { services } from '../../../config/services'

import ApplicationLogo from './ApplicationLogo'
import MainNavigation from '../navigation/MainNavigation';
import SearchAndCTA from '../navigation/SearchAndCTA';

// Define menu item types
interface SubMenuItem {
    label: string;
    href: string;
}

interface MenuItem {
    title: string;
    href: string;
    subMenu?: SubMenuItem[];
}

// Generate services menu items from config
const generateServicesMenu = (): MenuItem[] => {
    const servicesSubMenu: SubMenuItem[] = Object.values(services).map(service => ({
        label: service.name,
        href: service.href
    }));

    return [
        {
            title: 'Hash Services',
            href: '/services',
            subMenu: servicesSubMenu
        },
        {
            title: 'API Docs',
            href: '/docs'
        },
        {
            title: 'Examples',
            href: '/examples'
        },
        {
            title: 'Security',
            href: '/security'
        },
        {
            title: 'Pricing',
            href: '/pricing'
        },
        {
            title: 'Blog',
            href: '/blog'
        },
        {
            title: 'Support',
            href: '/support'
        }
    ];
};

const MidBarNav = () => {
    const menuItems = generateServicesMenu();

    return (
        <div className="w-full bg-gray-900/95 backdrop-blur-md border-b border-gray-700/50 relative z-50">
            <div className="w-full max-w-7xl mx-auto px-6 py-4">
                <div className="flex justify-between items-center">
                    {/* Logo Section */}
                    <ApplicationLogo logo="CodexHash" showIcon={false} />

                    {/* Navigation Menu */}
                    <MainNavigation menuItems={menuItems} />

                    {/* Right Section - Search & CTA */}
                    <SearchAndCTA />
                </div>
            </div>
        </div>
    )
}

export default MidBarNav
