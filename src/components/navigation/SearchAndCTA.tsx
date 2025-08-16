import Link from 'next/link'
import React from 'react'

const SearchAndCTA = () => {
  return (
      <div className="flex items-center gap-4">
          {/* Search Icon */}
          <button className="p-2 text-gray-400 hover:text-blue-400 transition-colors duration-300 rounded-lg hover:bg-gray-800/50">
              <i className="ph ph-magnifying-glass text-lg"></i>
          </button>

          {/* CTA Button */}
          <Link
              href="/contact"
              className="px-6 py-2.5 bg-gradient-to-r from-blue-600 to-purple-600 text-white font-business font-medium text-sm rounded-lg hover:from-blue-700 hover:to-purple-700 transition-all duration-300 shadow-md hover:shadow-lg"
          >
              Get A Free Quote
          </Link>

          {/* Mobile Menu Button */}
          <button className="lg:hidden p-2 text-white hover:text-blue-400 transition-colors duration-300">
              <i className="ph ph-list text-xl"></i>
          </button>
      </div>
  )
}

export default SearchAndCTA