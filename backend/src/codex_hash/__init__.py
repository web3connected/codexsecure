"""
CodexHash - Python Implementation
=================================

A clean, high-performance Python implementation of the CodexHash algorithm
for domain-specific hashing with harmonic frequency calculations.

This implementation focuses on:
- Mathematical precision with NumPy
- Performance optimization
- Clean API interfaces
- Comprehensive testing

Note: This is a domain-specific hash function, not a cryptographic primitive.
For security-critical applications, use established algorithms like Argon2id or SHA-3.
"""

from .core import CodexHarmonicHash, CodexHashResult
from .time_utils import CodexTime, TIUCalculator
from .interfaces import HashFunction, TimeProvider, TestMode, BenchmarkableHash
from .exceptions import (
    CodexHashError, InvalidTIUError, HashValidationError,
    EntropyCalculationError, HarmonicFrequencyError, TimeProviderError
)

__version__ = "1.0.0"
__author__ = "Web3Connected"

__all__ = [
    "CodexHarmonicHash",
    "CodexHashResult", 
    "CodexTime",
    "TIUCalculator",
    "HashFunction",
    "TimeProvider",
    "TestMode",
    "BenchmarkableHash",
    "CodexHashError",
    "InvalidTIUError", 
    "HashValidationError",
    "EntropyCalculationError",
    "HarmonicFrequencyError",
    "TimeProviderError"
]
