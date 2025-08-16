'use client'

import React, { useEffect, useState } from 'react'
import { RotateCcw, Clock, Infinity } from 'lucide-react'

const CorePrinciples = () => {
    const [scrollY, setScrollY] = useState(0);

    useEffect(() => {
        const handleScroll = () => {
            setScrollY(window.scrollY);
        };

        window.addEventListener('scroll', handleScroll);
        return () => window.removeEventListener('scroll', handleScroll);
    }, []);

    return (
        <section className="relative py-20  overflow-hidden " style={{
            height: "650px"
        }}>
            {/* Background image with parallax */}
            <div
                className="absolute inset-0 bg-cover bg-center bg-no-repeat opacity-100"
                style={{
                    backgroundImage: "url('/assets/images/360_F_471333224_bQbXMG1TuzIed78bfWp2UVLN1kyhoQYf.jpg')",
                    transform: `translateY(${scrollY * 0.5}px)`,
                    backgroundAttachment: 'fixed',
                    backgroundSize: '120%',
                    willChange: 'transform'
                }}
            />
            {/* Lighter overlay for better background visibility */}
            <div className="absolute inset-0 bg-gradient-to-br from-slate-900/70 via-blue-900/60 to-slate-900/70"></div>
            
            <div className="relative container mx-auto px-6 text-center">
                {/* Title */}
                <h2 className="text-4xl font-bold text-white mb-12">
                    Core Principles of the Codex
                </h2>

                {/* 3 Panel Grid */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-6xl mx-auto">
                    {/* Inversion */}
                    <div className="p-8 bg-gradient-to-br from-blue-900/40 to-slate-800 rounded-2xl border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300">
                        <div className="w-16 h-16 mx-auto mb-6 flex items-center justify-center rounded-xl bg-blue-500/20">
                            <RotateCcw className="w-8 h-8 text-blue-400" />
                        </div>
                        <h3 className="text-2xl font-semibold text-white mb-4">Inversion</h3>
                        <p className="text-slate-300 leading-relaxed">
                            The foundational principle of harmonic reality, where structures emerge from
                            mirrored opposites, generating stability through balance.
                        </p>
                    </div>

                    {/* Time */}
                    <div className="p-8 bg-gradient-to-br from-purple-900/40 to-slate-800 rounded-2xl border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300">
                        <div className="w-16 h-16 mx-auto mb-6 flex items-center justify-center rounded-xl bg-purple-500/20">
                            <Clock className="w-8 h-8 text-purple-400" />
                        </div>
                        <h3 className="text-2xl font-semibold text-white mb-4">Time</h3>
                        <p className="text-slate-300 leading-relaxed">
                            Not a dimension, but a rhythm — the refresh rate of the universe that synchronizes
                            all motion and experience.
                        </p>
                    </div>

                    {/* Totality (12) */}
                    <div className="p-8 bg-gradient-to-br from-green-900/40 to-slate-800 rounded-2xl border border-white/10 shadow-lg hover:shadow-xl transition-all duration-300">
                        <div className="w-16 h-16 mx-auto mb-6 flex items-center justify-center rounded-xl bg-green-500/20">
                            <Infinity className="w-8 h-8 text-green-400" />
                        </div>
                        <h3 className="text-2xl font-semibold text-white mb-4">Totality (12)</h3>
                        <p className="text-slate-300 leading-relaxed">
                            The harmonic cycle of twelve zones — the complete framework through which existence
                            organizes itself into wholeness.
                        </p>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default CorePrinciples
