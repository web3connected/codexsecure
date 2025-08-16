'use client'

import React from 'react'
import Link from 'next/link'
import Image from 'next/image'

interface User {
  name: string
  avatar?: string
  role?: string
  id: string
}

interface WhosLoggedInProps {
  user?: User
}

const WhosLoggedIn: React.FC<WhosLoggedInProps> = ({ user }) => {
    const isLoggedIn = Boolean(user) // replace with your real auth check

    if (!isLoggedIn || !user) {
        return (
            <div className="flex items-center gap-2">
                <Link
                    href="/login"
                    className="px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/80 transition"
                >
                    Login
                </Link>
                <Link
                    href="/register"
                    className="px-3 py-1.5 text-sm rounded-lg border border-primary text-primary hover:bg-primary hover:text-white transition"
                >
                    Register
                </Link>
            </div>
        )
    }

    return (
        <div className="relative group">

            {/* Notifications */}
            <button className="relative p-1.5 theme-text-muted hover:theme-accent transition-colors rounded-lg hover:theme-bg-surface">
                <i className="ph ph-bell text-base"></i>
                <span className="absolute -top-0.5 -right-0.5 w-4 h-4 bg-red-500 text-white text-xs rounded-full flex items-center justify-center">
                    3
                </span>
            </button>

            {/* Messages */}
            <button className="relative p-1.5 theme-text-muted hover:theme-accent transition-colors rounded-lg hover:theme-bg-surface">
                <i className="ph ph-chat-circle text-base"></i>
                <span className="absolute -top-0.5 -right-0.5 w-4 h-4 theme-bg-primary text-white text-xs rounded-full flex items-center justify-center">
                    5
                </span>
            </button>

            <button className="flex items-center gap-2 focus:outline-none">
                <Image
                    src={user.avatar || '/assets/images/default-avatar.png'}
                    alt={user.name}
                    width={28}
                    height={28}
                    className="rounded-full border border-gray-300"
                />
                <span className="hidden sm:inline text-sm">{user.name}</span>
            </button>

            {/* Dropdown */}
            <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border opacity-0 group-hover:opacity-100 pointer-events-none group-hover:pointer-events-auto transition">
                <Link
                    href="/account"
                    className="block px-4 py-2 hover:bg-gray-100"
                >
                    My Account
                </Link>
                <Link
                    href="/settings"
                    className="block px-4 py-2 hover:bg-gray-100"
                >
                    Settings
                </Link>
                <button
                    onClick={() => console.log('Logout')}
                    className="w-full text-left px-4 py-2 hover:bg-gray-100"
                >
                    Logout
                </button>
            </div>
        </div>
    )
}

export default WhosLoggedIn
