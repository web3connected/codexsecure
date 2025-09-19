'use client'

import React, { createContext, useContext, useState, useEffect } from 'react'
import PageLoader from '@/components/widgets/ui/PageLoader'

interface LoadingContextType {
  isLoading: boolean
  setLoading: (loading: boolean) => void
  showLoader: (message?: string) => void
  hideLoader: () => void
  message: string
}

const LoadingContext = createContext<LoadingContextType | undefined>(undefined)

export const useLoading = () => {
  const context = useContext(LoadingContext)
  if (!context) {
    throw new Error('useLoading must be used within a LoadingProvider')
  }
  return context
}

interface LoadingProviderProps {
  children: React.ReactNode
  initialLoading?: boolean
  initialMessage?: string
}

export const LoadingProvider: React.FC<LoadingProviderProps> = ({ 
  children, 
  initialLoading = true,
  initialMessage = "Initializing Web3 Codex..."
}) => {
  const [isLoading, setIsLoading] = useState(initialLoading)
  const [message, setMessage] = useState(initialMessage)

  // Auto-hide initial loader after 2 seconds
  useEffect(() => {
    if (initialLoading) {
      const timer = setTimeout(() => {
        setIsLoading(false)
      }, 2000)
      return () => clearTimeout(timer)
    }
  }, [initialLoading])

  const setLoading = (loading: boolean) => {
    setIsLoading(loading)
  }

  const showLoader = (newMessage?: string) => {
    if (newMessage) {
      setMessage(newMessage)
    }
    setIsLoading(true)
  }

  const hideLoader = () => {
    setIsLoading(false)
  }

  const value = {
    isLoading,
    setLoading,
    showLoader,
    hideLoader,
    message
  }

  return (
    <LoadingContext.Provider value={value}>
      {children}
      {isLoading && <PageLoader message={message} />}
    </LoadingContext.Provider>
  )
}
