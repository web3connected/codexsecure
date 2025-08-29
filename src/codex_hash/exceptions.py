"""
CodexHash Exception Classes
==========================

Custom exceptions for CodexHash operations, designed for comprehensive
error handling in testing and production environments.
"""

from typing import Optional, Any, Dict


class CodexHashError(Exception):
    """Base exception for all CodexHash operations."""
    
    def __init__(self, message: str, details: Optional[Dict[str, Any]] = None):
        super().__init__(message)
        self.details = details or {}
        
    def to_dict(self) -> Dict[str, Any]:
        """Convert exception to dictionary for JSON serialization."""
        return {
            'error': self.__class__.__name__,
            'message': str(self),
            'details': self.details
        }


class InvalidTIUError(CodexHashError):
    """Raised when TIU (Time Integrity Unit) value is invalid."""
    
    def __init__(self, tiu_value: float, reason: str):
        self.tiu_value = tiu_value
        super().__init__(
            f"Invalid TIU value {tiu_value}: {reason}",
            {'tiu_value': tiu_value, 'reason': reason}
        )


class HashValidationError(CodexHashError):
    """Raised when hash validation fails."""
    
    def __init__(self, expected: str, actual: str, context: str = ""):
        self.expected = expected
        self.actual = actual
        self.context = context
        super().__init__(
            f"Hash validation failed{': ' + context if context else ''}. "
            f"Expected: {expected}, Got: {actual}",
            {'expected': expected, 'actual': actual, 'context': context}
        )


class EntropyCalculationError(CodexHashError):
    """Raised when entropy calculations fail."""
    pass


class HarmonicFrequencyError(CodexHashError):
    """Raised when harmonic frequency calculations fail."""
    pass


class TimeProviderError(CodexHashError):
    """Raised when time provider operations fail."""
    pass


class BenchmarkError(CodexHashError):
    """Raised during benchmark operations."""
    pass


class KATError(CodexHashError):
    """Raised during Known Answer Test operations."""
    pass
