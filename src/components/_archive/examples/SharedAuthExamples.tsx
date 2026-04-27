/**
 * Example: Using Shared Database Auth in Components
 * This shows how simple it is to add authentication to any component
 */

'use client'

import { useAuth } from '@/hooks'
import { getSiteDisplayName } from '@/types/shared'

export function UserProfileExample() {
  const { user, isAuthenticated, currentSite } = useAuth()
  
  if (!isAuthenticated) {
    return <div>Please log in</div>
  }
  
  return (
    <div className="p-4 bg-slate-800 rounded-lg">
      <h2 className="text-xl font-bold mb-4">User Profile</h2>
      
      <div className="space-y-2">
        <p><strong>Name:</strong> {user?.name}</p>
        <p><strong>Email:</strong> {user?.email}</p>
        <p><strong>Current Site:</strong> {getSiteDisplayName(currentSite)}</p>
        
        <div className="mt-4">
          <p className="font-semibold mb-2">Registered Sites:</p>
          <div className="flex gap-2">
            {user?.reg_sites?.map(site => (
              <span 
                key={site}
                className="px-3 py-1 bg-blue-600 rounded-full text-sm"
              >
                {getSiteDisplayName(site)}
              </span>
            ))}
          </div>
        </div>
        
        {user?.isRegisteredOnCurrentSite ? (
          <p className="text-green-400 mt-4">
            ✓ You are registered on {getSiteDisplayName(currentSite)}
          </p>
        ) : (
          <p className="text-yellow-400 mt-4">
            First time visiting {getSiteDisplayName(currentSite)}!
          </p>
        )}
      </div>
    </div>
  )
}

/**
 * Example: Protected Route Component
 * Simple check for authentication
 */
export function ProtectedContent({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  
  if (isLoading) {
    return <div>Loading...</div>
  }
  
  if (!isAuthenticated) {
    return <div>Access Denied. Please log in.</div>
  }
  
  return <>{children}</>
}

/**
 * Example: Conditional Rendering Based on Registration
 * Show different content based on which sites user has registered
 */
export function CrossSiteFeature() {
  const { user, isAuthenticated } = useAuth()
  
  if (!isAuthenticated) return null
  
  const hasWeb3Connected = user?.reg_sites?.includes('web3connected') ?? false
  const hasCodexHash = user?.reg_sites?.includes('codexhash') ?? false
  const hasCodexSecure = user?.reg_sites?.includes('codexsecure') ?? false
  
  return (
    <div className="p-4 border border-slate-700 rounded-lg">
      <h3 className="font-bold mb-2">Your Codex Ecosystem Access</h3>
      
      <ul className="space-y-2">
        <li className={hasWeb3Connected ? 'text-green-400' : 'text-gray-500'}>
          {hasWeb3Connected ? '✓' : '○'} Web3 Connected
        </li>
        <li className={hasCodexHash ? 'text-green-400' : 'text-gray-500'}>
          {hasCodexHash ? '✓' : '○'} CodexHash
        </li>
        <li className={hasCodexSecure ? 'text-green-400' : 'text-gray-500'}>
          {hasCodexSecure ? '✓' : '○'} CodexSecure
        </li>
      </ul>
      
      {(user?.reg_sites?.length ?? 0) === 3 && (
        <p className="mt-4 text-purple-400 font-semibold">
          🎉 You have full ecosystem access!
        </p>
      )}
    </div>
  )
}

/**
 * That's it! The shared database architecture makes it simple:
 * 
 * 1. Import useAuth hook
 * 2. Destructure what you need
 * 3. Use the data
 * 
 * No complex logic, no manual database connections, no syncing needed.
 * The backend handles all the reg_sites updates automatically.
 */
