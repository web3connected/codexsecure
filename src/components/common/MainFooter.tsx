import React from 'react'
import Link from 'next/link'
import Image from 'next/image'
import { services } from '../../../config/services'
import { MENU } from '../../data/mock/menunav'

const MainFooter = () => {
    return (
        <footer className="bg-gray-900/95 backdrop-blur-md border-t border-gray-700/50 mt-auto">
            {/* Main Footer Content */}
            <div className="w-full max-w-7xl mx-auto px-6 py-12" >
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-8">
                    {/* Company Info */}
                    <div className="lg:col-span-1">
                        <Link href={MENU.brand.href} className="flex items-center gap-3 mb-4">
                            <Image 
                                src="/assets/icons/codexIcon.png" 
                                width={32} 
                                height={32} 
                                className="rounded" 
                                alt={`${MENU.brand.name} Logo`} 
                            />
                            <span className="font-heading font-bold text-xl text-white">
                                {MENU.brand.name.split(' ')[0]} <span className="text-blue-400">{MENU.brand.accent || MENU.brand.name.split(' ')[1]}</span>
                            </span>
                        </Link>
                        <p className="text-gray-400 text-sm mb-4 max-w-xs">
                            Professional blockchain solutions, wallet connections, and DeFi integrations tailored for government and Fortune 500 companies.
                        </p>
                        <div className="flex items-center gap-4">
                            {MENU.footer.socials?.map((social, index) => (
                                <Link 
                                    key={index}
                                    href={social.href} 
                                    className="text-gray-400 hover:text-blue-400 transition-colors"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    aria-label={social.label}
                                >
                                    <i className={`${social.icon} text-xl`}></i>
                                </Link>
                            ))}
                        </div>
                    </div>

                    {/* Services - Dynamic from config */}
                    <div>
                        <h3 className="font-business font-semibold text-white mb-4">Services</h3>
                        <ul className="space-y-3">
                            {Object.values(services).map((service, index) => (
                                <li key={index}>
                                    <Link href={service.href} className="text-gray-400 hover:text-blue-400 transition-colors text-sm">
                                        {service.name}
                                    </Link>
                                </li>
                            ))}
                        </ul>
                    </div>

                    {/* Company - Dynamic from config */}
                    <div>
                        <h3 className="font-business font-semibold text-white mb-4">
                            {MENU.footer.columns[0].title}
                        </h3>
                        <ul className="space-y-3">
                            {MENU.footer.columns[0].items.map((item, index) => (
                                <li key={index}>
                                    <Link 
                                        href={item.href || '#'} 
                                        className="text-gray-400 hover:text-blue-400 transition-colors text-sm inline-flex items-center gap-2"
                                        {...(item.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                    >
                                        {item.label}
                                        {item.badge && (
                                            <span className={`px-2 py-0.5 text-xs rounded-full ${
                                                item.badge.tone === 'primary' ? 'bg-blue-500/20 text-blue-400' :
                                                item.badge.tone === 'danger' ? 'bg-red-500/20 text-red-400' :
                                                'bg-gray-500/20 text-gray-400'
                                            }`}>
                                                {item.badge.text}
                                            </span>
                                        )}
                                    </Link>
                                </li>
                            ))}
                        </ul>
                    </div>

                    {/* Resources - Dynamic from config */}
                    <div>
                        <h3 className="font-business font-semibold text-white mb-4">
                            {MENU.footer.columns[2].title}
                        </h3>
                        <ul className="space-y-3">
                            {MENU.footer.columns[2].items.map((item, index) => (
                                <li key={index}>
                                    <Link 
                                        href={item.href || '#'} 
                                        className="text-gray-400 hover:text-blue-400 transition-colors text-sm"
                                        {...(item.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                    >
                                        {item.label}
                                    </Link>
                                </li>
                            ))}
                        </ul>
                    </div>

                    {/* Account - Dynamic from config */}
                    <div>
                        <h3 className="font-business font-semibold text-white mb-4">
                            {MENU.footer.columns[3].title}
                        </h3>
                        <ul className="space-y-3">
                            {MENU.footer.columns[3].items.slice(0, 4).map((item, index) => (
                                <li key={index}>
                                    <Link 
                                        href={item.href || '#'} 
                                        className="text-gray-400 hover:text-blue-400 transition-colors text-sm"
                                        {...(item.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                    >
                                        {item.label}
                                    </Link>
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
            </div>

            {/* Bottom Footer */}
            <div className="border-t border-gray-700/50">
                <div className="w-full max-w-7xl mx-auto px-6 py-6">
                    <div className="flex flex-col md:flex-row justify-between items-center gap-4">
                        <div className="flex flex-col md:flex-row items-center gap-4 text-sm text-gray-400">
                            <p>&copy; 2025 {MENU.brand.name}. All rights reserved.</p>
                            <div className="flex items-center gap-4">
                                {MENU.footer.bottom.map((item, index) => (
                                    <Link 
                                        key={index}
                                        href={item.href || '#'} 
                                        className="hover:text-blue-400 transition-colors inline-flex items-center gap-1"
                                        {...(item.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                                    >
                                        {item.icon && <i className={item.icon}></i>}
                                        {item.label}
                                    </Link>
                                ))}
                            </div>
                        </div>
                        <div className="flex items-center gap-2 text-sm text-gray-400">
                            <i className="ph ph-map-pin text-base"></i>
                            <span>Global Professional Services</span>
                        </div>
                    </div>
                </div>
            </div>
        </footer>
    )
}

export default MainFooter
