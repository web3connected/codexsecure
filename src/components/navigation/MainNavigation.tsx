"use client"

import Link from 'next/link'
import React, { useState, useEffect, useRef } from 'react'

interface MenuItem {
    title: string;
    href: string;
    subMenu?: {
        label: string;
        href: string;
    }[];
}

const MainNavigation = ({
    menuItems
}: {
    menuItems: MenuItem[];
}) => {
    const [openDropdown, setOpenDropdown] = useState<number | null>(null);
    const [hoverTimeout, setHoverTimeout] = useState<NodeJS.Timeout | null>(null);
    const navRef = useRef<HTMLDivElement>(null);

    const handleMenuClick = (idx: number, hasSubMenu: boolean) => {
        if (hasSubMenu) {
            // Clear any existing timeout
            if (hoverTimeout) {
                clearTimeout(hoverTimeout);
                setHoverTimeout(null);
            }
            setOpenDropdown(openDropdown === idx ? null : idx);
        }
    };

    const handleMouseEnter = (idx: number) => {
        // Clear any existing timeout
        if (hoverTimeout) {
            clearTimeout(hoverTimeout);
            setHoverTimeout(null);
        }
        setOpenDropdown(idx);
    };

    const handleMouseLeave = () => {
        // Use a timeout to allow user to move to dropdown
        const timeout = setTimeout(() => {
            setOpenDropdown(null);
        }, 150); // 150ms delay
        setHoverTimeout(timeout);
    };

    const clearHoverTimeout = () => {
        if (hoverTimeout) {
            clearTimeout(hoverTimeout);
            setHoverTimeout(null);
        }
    };

    // Close dropdown when clicking outside
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (navRef.current && !navRef.current.contains(event.target as Node)) {
                setOpenDropdown(null);
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
            // Clean up timeout on unmount
            if (hoverTimeout) {
                clearTimeout(hoverTimeout);
            }
        };
    }, [hoverTimeout]);

  return (
      <nav className="hidden lg:block" ref={navRef}>
          <ul className="flex items-center gap-8">
              {menuItems.map((menu, idx) => (
                  <li 
                      key={idx} 
                      className="relative"
                      onMouseLeave={handleMouseLeave}
                  >
                      <div className="flex items-center gap-1 py-2">
                          {menu.subMenu ? (
                              <button
                                  onClick={() => handleMenuClick(idx, true)}
                                  onMouseEnter={() => handleMouseEnter(idx)}
                                  className="flex items-center gap-1 font-business text-sm font-medium text-white hover:text-blue-400 transition-colors duration-300 cursor-pointer"
                              >
                                  {menu.title}
                                  <i className={`ph ph-caret-down text-sm text-gray-400 transition-transform duration-300 ml-1 ${
                                      openDropdown === idx ? 'rotate-180' : ''
                                  }`} />
                              </button>
                          ) : (
                              <Link
                                  href={menu.href}
                                  className="font-business text-sm font-medium text-white hover:text-blue-400 transition-colors duration-300"
                              >
                                  {menu.title}
                              </Link>
                          )}
                      </div>

                      {/* Dropdown Menu */}
                      {menu.subMenu && (
                          <div className="absolute top-full left-0 pt-1 z-[9999]">
                              <ul 
                                  className={`w-56 bg-gray-800/95 backdrop-blur-md border border-gray-700/50 rounded-lg shadow-lg transition-all duration-300 transform z-[9999] ${
                                      openDropdown === idx 
                                          ? 'opacity-100 visible translate-y-0' 
                                          : 'opacity-0 invisible translate-y-2'
                                  }`}
                                  onMouseEnter={clearHoverTimeout}
                                  onMouseLeave={handleMouseLeave}
                              >
                                  <div className="py-2">
                                      {menu.subMenu.map((sub, subIdx) => (
                                          <li key={subIdx}>
                                              <Link
                                                  href={sub.href}
                                                  className="block px-4 py-2 font-business text-sm text-gray-300 hover:bg-gray-700/50 hover:text-blue-400 transition-colors duration-200"
                                                  onClick={() => setOpenDropdown(null)}
                                              >
                                                  {sub.label}
                                              </Link>
                                          </li>
                                      ))}
                                  </div>
                              </ul>
                          </div>
                      )}
                  </li>
              ))}
          </ul>
      </nav>
  )
}

export default MainNavigation