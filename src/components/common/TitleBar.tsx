import Link from 'next/link';
import React from 'react';

interface BreadcrumbItem {
    label: string;
    href?: string;
}

interface TopBarProps {
    title: string;
    description: string;
    breadcrumbs?: BreadcrumbItem[];
}

const TopBar: React.FC<TopBarProps> = ({
    title,
    description,
    breadcrumbs = [{ label: 'Home', href: '/' }],
}) => {
    // Ensure the trail ends with the current page title (non-link)
    let trail: BreadcrumbItem[] = breadcrumbs.length
        ? [...breadcrumbs]
        : [{ label: 'Home', href: '/' }];

    const last = trail[trail.length - 1];
    if (!last || last.label !== title) {
        trail = [...trail, { label: title }];
    }

    return (
        <section className="relative py-20 px-6">
            <div className="max-w-7xl mx-auto">
                <div className="text-center mb-16">
                    <h1 className="text-5xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-blue-400 via-purple-500 to-cyan-400 bg-clip-text text-transparent">
                        {title}
                    </h1>
                    <p className="text-xl text-gray-300 max-w-3xl mx-auto leading-relaxed">
                        {description}
                    </p>
                </div>

                {/* Dynamic Navigation Breadcrumb */}
                <nav className="mb-12" aria-label="Breadcrumb">
                    <ol className="flex items-center text-sm text-gray-400">
                        {trail.map((item, index) => (
                            <React.Fragment key={`${item.href ?? item.label}-${index}`}>
                                {index > 0 && (
                                    <li aria-hidden="true" className="px-2">
                                        /
                                    </li>
                                )}
                                <li>
                                    {item.href ? (
                                        <Link href={item.href} className="hover:text-blue-400 transition-colors">
                                            {item.label}
                                        </Link>
                                    ) : (
                                        <span className="text-blue-400">{item.label}</span>
                                    )}
                                </li>
                            </React.Fragment>
                        ))}
                    </ol>
                </nav>
            </div>
        </section>
    );
};

export default TopBar;
