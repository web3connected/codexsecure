import React from 'react'
import TopBarNav from './TopBarNav'
import MidBarNav from './MidBarNav'

const MainHeader = () => {
    return (
        <header className="top-0 left-0 right-0 z-50 header nav-theme w-full">           
            <TopBarNav />
            <MidBarNav />
        </header>
    )
}

export default MainHeader
