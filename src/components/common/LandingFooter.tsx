import React from 'react'
import { Button } from '../widgets/ui/button'

const LandingFooter = () => {
  return (
    <footer id="contact" className="bg-gray-900 border-t border-gray-800 py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center">
            <div className="flex items-center space-x-4 mb-4 md:mb-0">
              <div className="w-10 h-10 bg-gradient-to-r from-purple-500 to-blue-500 rounded-lg flex items-center justify-center">
                <span className="font-bold text-white">C</span>
              </div>
              <div>
                <h3 className="text-lg font-bold">CodexHash</h3>
                <p className="text-sm text-gray-400">Quantum-Resistant Technology</p>
              </div>
            </div>
            
            <div className="text-center">
              <p className="text-gray-400 mb-2">Ready to implement quantum-resistant hashing in your application?</p>
              <Button className="bg-gradient-to-r from-purple-500 to-blue-500">
                Contact: contact@web3connected.com
              </Button>
            </div>
          </div>
          
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-500">
            <p>&copy; 2024 Web3Connected. All rights reserved.</p>
          </div>
        </div>
      </footer>
  )
}

export default LandingFooter