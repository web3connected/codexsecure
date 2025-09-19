import React, { useEffect, useState } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import { Button } from '@/components/widgets/ui/button'
import Link from 'next/link'

const LandingHeader = () => {
  const [activeSection, setActiveSection] = useState('security')
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    // Add smooth scrolling to the document
    const originalStyle = document.documentElement.style.scrollBehavior
    document.documentElement.style.scrollBehavior = 'smooth'

    // Handle hash navigation when coming from other pages
    if (pathname === '/' && window.location.hash) {
      const targetId = window.location.hash.substring(1)
      setTimeout(() => {
        const element = document.getElementById(targetId)
        if (element) {
          element.scrollIntoView({
            behavior: 'smooth',
            block: 'start'
          })
          setActiveSection(targetId)
        }
      }, 100) // Small delay to ensure page is loaded
    }

    // Only set up intersection observer on the home page
    if (pathname === '/') {
      // Create intersection observer to track active section
      const sections = ['security', 'features', 'demo', 'technology', 'contact']
      const observer = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              setActiveSection(entry.target.id)
            }
          })
        },
        {
          threshold: 0.5,
          rootMargin: '-100px 0px -50% 0px'
        }
      )

      // Observe all sections
      sections.forEach((id) => {
        const element = document.getElementById(id)
        if (element) {
          observer.observe(element)
        }
      })

      return () => {
        document.documentElement.style.scrollBehavior = originalStyle
        observer.disconnect()
      }
    }

    return () => {
      document.documentElement.style.scrollBehavior = originalStyle
    }
  }, [pathname])

  const handleNavClick = (e: React.MouseEvent<HTMLAnchorElement>, targetId: string) => {
    e.preventDefault()
    
    // If we're not on the home page, navigate to home first
    if (pathname !== '/') {
      router.push(`/#${targetId}`)
      return
    }

    // If we're on the home page, scroll to the section
    const element = document.getElementById(targetId)
    if (element) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      })
    }
  }

  const handleLogoClick = (e: React.MouseEvent) => {
    e.preventDefault()
    if (pathname !== '/') {
      router.push('/')
    } else {
      // Scroll to top if already on home page
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }
  return (
    <header className="fixed top-0 w-full z-50 bg-gray-900/95 backdrop-blur-sm border-b border-gray-800">
           <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
             <div className="flex justify-between items-center py-6">
               <button 
                 onClick={handleLogoClick}
                 className="flex items-center space-x-4 hover:opacity-80 transition-opacity cursor-pointer"
               >
                 <div className="w-10 h-10 bg-gradient-to-r from-purple-500 to-blue-500 rounded-lg flex items-center justify-center">
                   <span className="font-bold text-white">C</span>
                 </div>
                 <div>
                   <h1 className="text-xl font-bold">CodexHash</h1>
                   <p className="text-xs text-gray-400">Quantum-Resistant Technology</p>
                 </div>
               </button>
               
               <nav className="hidden md:flex space-x-8">
                 <a 
                   href="#features" 
                   onClick={(e) => handleNavClick(e, 'features')}
                   className={`transition-all duration-300 relative ${
                     activeSection === 'features' 
                       ? 'text-purple-400 transform scale-105' 
                       : 'text-gray-300 hover:text-white hover:transform hover:scale-105'
                   }`}
                 >
                   Features
                   {activeSection === 'features' && (
                     <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-purple-500 to-blue-500 animate-pulse"></div>
                   )}
                 </a>
                 <a 
                   href="#demo" 
                   onClick={(e) => handleNavClick(e, 'demo')}
                   className={`transition-all duration-300 relative ${
                     activeSection === 'demo' 
                       ? 'text-purple-400 transform scale-105' 
                       : 'text-gray-300 hover:text-white hover:transform hover:scale-105'
                   }`}
                 >
                   Demo
                   {activeSection === 'demo' && (
                     <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-purple-500 to-blue-500 animate-pulse"></div>
                   )}
                 </a>
                 <a 
                   href="#security" 
                   onClick={(e) => handleNavClick(e, 'security')}
                   className={`transition-all duration-300 relative ${
                     activeSection === 'security' 
                       ? 'text-purple-400 transform scale-105' 
                       : 'text-gray-300 hover:text-white hover:transform hover:scale-105'
                   }`}
                 >
                   Security
                   {activeSection === 'security' && (
                     <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-purple-500 to-blue-500 animate-pulse"></div>
                   )}
                 </a>
                 <a 
                   href="#technology" 
                   onClick={(e) => handleNavClick(e, 'technology')}
                   className={`transition-all duration-300 relative ${
                     activeSection === 'technology' 
                       ? 'text-purple-400 transform scale-105' 
                       : 'text-gray-300 hover:text-white hover:transform hover:scale-105'
                   }`}
                 >
                   Technology
                   {activeSection === 'technology' && (
                     <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-purple-500 to-blue-500 animate-pulse"></div>
                   )}
                 </a>
                 <a 
                   href="#contact" 
                   onClick={(e) => handleNavClick(e, 'contact')}
                   className={`transition-all duration-300 relative ${
                     activeSection === 'contact' 
                       ? 'text-purple-400 transform scale-105' 
                       : 'text-gray-300 hover:text-white hover:transform hover:scale-105'
                   }`}
                 >
                   Contact
                   {activeSection === 'contact' && (
                     <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-purple-500 to-blue-500 animate-pulse"></div>
                   )}
                 </a>
               </nav>
               
               <Button className="bg-gradient-to-r from-purple-500 to-blue-500 hover:opacity-90">
                 Get Early Access
               </Button>
             </div>
           </div>
         </header>
  )
}

export default LandingHeader