'use client'

import React, { createContext, useContext, useState, useEffect } from 'react'
import type { User, UserWithSiteContext, CodexSite } from '@/types/shared'
import { getCurrentSite, isUserRegisteredOnSite } from '@/types/shared'

interface AuthContextType {
  user: UserWithSiteContext | null
  isAuthenticated: boolean
  isLoading: boolean
  currentSite: CodexSite
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  checkAuth: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: React.ReactNode
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<UserWithSiteContext | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isMounted, setIsMounted] = useState(false)
  const currentSite = getCurrentSite()

  // Prevent hydration mismatch
  useEffect(() => {
    setIsMounted(true)
  }, [])

  // Check authentication on mount
  useEffect(() => {
    if (isMounted) {
      checkAuth()
    }
  }, [isMounted])

  const checkAuth = async () => {
    try {
      setIsLoading(true)
      
      // Check for stored auth token/session
      const token = localStorage.getItem('auth_token')
      if (token) {
        // In production, validate token with backend and get user data
        // For now, use mock data that matches shared DB structure
        const mockUser: User = {
          id: '1',
          email: 'user@example.com',
          name: 'Demo User',
          reg_sites: ['web3connected', 'codexsecure'], // Registered on multiple sites
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          last_login_at: new Date().toISOString(),
          last_login_site: currentSite,
          profile_data: {},
          preferences: {}
        }
        
        // Add site context
        const userWithContext: UserWithSiteContext = {
          ...mockUser,
          isRegisteredOnCurrentSite: isUserRegisteredOnSite(mockUser, currentSite),
          currentSite
        }
        
        setUser(userWithContext)
      }
    } catch (error) {
      console.error('Auth check failed:', error)
      setUser(null)
    } finally {
      setIsLoading(false)
    }
  }

  const login = async (email: string, password: string) => {
    try {
      setIsLoading(true)
      
      // In production, call authentication API
      // POST /api/auth/login { email, password, currentSite }
      // Backend will add currentSite to reg_sites if not present
      await new Promise(resolve => setTimeout(resolve, 500))
      
      const mockUser: User = {
        id: '1',
        email,
        name: 'Demo User',
        reg_sites: [currentSite], // New user, only registered on current site
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        last_login_at: new Date().toISOString(),
        last_login_site: currentSite,
        profile_data: {},
        preferences: {}
      }
      
      const userWithContext: UserWithSiteContext = {
        ...mockUser,
        isRegisteredOnCurrentSite: true,
        currentSite
      }
      
      localStorage.setItem('auth_token', 'mock-token')
      setUser(userWithContext)
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    } finally {
      setIsLoading(false)
    }
  }

  const logout = async () => {
    try {
      setIsLoading(true)
      
      // In production, call logout API
      await new Promise(resolve => setTimeout(resolve, 300))
      
      localStorage.removeItem('auth_token')
      setUser(null)
    } catch (error) {
      console.error('Logout failed:', error)
      throw error
    } finally {
      setIsLoading(false)
    }
  }

  const value: AuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading: !isMounted || isLoading,
    currentSite,
    login,
    logout,
    checkAuth
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}
