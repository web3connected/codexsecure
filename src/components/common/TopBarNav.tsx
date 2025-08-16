'use client'

import React from 'react'
import DateGreeter from '../widgets/DateGreeter'
import WhosLoggedIn from '../profile/WhosLoggedIn'

const TopBarNav = () => {
    // Mock user data - replace with your actual auth state
    const mockUser = undefined;

    return (
        <div className="w-full bg-gray-900/95 backdrop-blur-md border-b border-gray-700/50">
            <div className="w-full max-w-none px-4 py-2">
                <div className="flex justify-between items-center w-full">
                    {/* Left Side - Date Greeter */}
                    <div className="flex items-center gap-3">
                        <div className="flex items-center gap-2 text-white">
                            <i className="ph ph-calendar text-base text-blue-400"></i>
                            <div className="font-business text-sm font-medium">
                                <DateGreeter />
                            </div>
                        </div>
                    </div>

                    {/* Right Side - User Profile, Messages, Notifications */}
                    <div className="flex items-center gap-2">
                        {/* Who's Logged In Component */}
                        <div className="ml-1">
                            <WhosLoggedIn user={mockUser} />
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default TopBarNav