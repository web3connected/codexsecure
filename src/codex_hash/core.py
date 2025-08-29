"""
CodexHash Core Implementation
============================

Pure Python implementation of the CodexHarmonicHash algorithm, converted from
the PHP implementation with comprehensive test framework support.

This implementation provides:
1. Full compatibility with existing test framework
2. Multiple test modes (deterministic, performance, cryptographic)
3. Comprehensive metadata for analysis
4. Mathematical precision optimized for benchmarking
5. Clean interfaces for MCP bridge integration
"""

import hashlib
import secrets
import time
import math
from typing import Optional, Dict, Any, List, Tuple
from dataclasses import dataclass, asdict

from .interfaces import (
    AbstractHashFunction, AbstractTimeProvider, TestMode, HashResult,
    BenchmarkableHash, CryptographicAnalyzer, KnownAnswerTestProvider
)
from .time_utils import CodexTime, TIUCalculator
from .exceptions import (
    CodexHashError, HashValidationError, InvalidTIUError,
    EntropyCalculationError, HarmonicFrequencyError
)


# Default configuration constants
DEFAULT_ROUNDS = 1000
DEFAULT_SALT_LENGTH = 32  # bytes
DEFAULT_BUFFER = 0.000001


@dataclass
class CodexHashResult(HashResult):
    """
    Extended hash result with CodexHash-specific metadata.
    
    This class extends the base HashResult to include additional
    metadata required for comprehensive testing and analysis.
    """
    
    # Additional CodexHash-specific fields
    entropy_modifier: float = 0.0
    harmonic_pressure: float = 0.0
    salt_entropy: float = 0.0
    calculation_metadata: Dict[str, Any] = None
    
    def __post_init__(self):
        """Initialize additional fields after dataclass creation."""
        if self.calculation_metadata is None:
            self.calculation_metadata = {}
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary with all metadata."""
        base_dict = super().to_dict()
        base_dict.update({
            'entropy_modifier': self.entropy_modifier,
            'harmonic_pressure': self.harmonic_pressure,
            'salt_entropy': self.salt_entropy,
            'calculation_metadata': self.calculation_metadata
        })
        return base_dict


class CodexHarmonicHash(AbstractHashFunction, BenchmarkableHash):
    """
    Pure Python implementation of CodexHarmonicHash algorithm.
    
    This implementation converts the PHP logic to Python with optimizations
    for testing, benchmarking, and mathematical precision.
    """
    
    def __init__(self, precision: int = 6):
        """
        Initialize CodexHarmonicHash.
        
        Args:
            precision: Decimal places for TIU calculations
        """
        self.precision = precision
        self.time_provider = CodexTime(precision)
        self.tiu_calculator = TIUCalculator(precision)
        
        # Cache for performance optimization
        self._base_frequency_cache = None
        self._last_salt = None
        self._last_tiu = None
        
        # Statistics for benchmarking
        self._stats = {
            'total_hashes': 0,
            'total_time_ns': 0,
            'avg_time_ns': 0.0
        }
    
    def set_test_mode(self, mode: TestMode, seed: Optional[int] = None) -> None:
        """
        Set test execution mode for this hasher instance.
        
        Args:
            mode: Test mode to use
            seed: Seed for deterministic modes
        """
        self.time_provider.set_test_mode(mode, seed)
    
    def get_base_frequency(self) -> float:
        """
        Get harmonic base frequency.
        
        Returns:
            Base frequency value from time provider
        """
        return self.time_provider.get_base_frequency()
    
    def _generate_salt(self, length: int = DEFAULT_SALT_LENGTH) -> str:
        """
        Generate cryptographically secure random salt.
        
        Args:
            length: Salt length in bytes
            
        Returns:
            Hex-encoded salt string
        """
        return secrets.token_hex(length)
    
    def _calculate_entropy_modifier(self, input_data: str, salt: str, tiu: float) -> float:
        """
        Calculate entropy modifier based on input characteristics.
        
        This implements the entropy calculation logic from the PHP version.
        
        Args:
            input_data: Input string
            salt: Salt string
            tiu: TIU value
            
        Returns:
            Entropy modifier value
        """
        try:
            # Combine input data characteristics
            input_length = len(input_data)
            salt_length = len(salt)
            
            # Calculate basic entropy components
            char_entropy = len(set(input_data)) / max(input_length, 1)
            salt_entropy = len(set(salt)) / max(salt_length, 1)
            
            # TIU-based entropy component
            tiu_entropy = math.log10(abs(tiu) + 1) / 20.0  # Normalized TIU contribution
            
            # Combine entropy components
            entropy_modifier = (char_entropy + salt_entropy + tiu_entropy) / 3.0
            
            return round(entropy_modifier, self.precision)
            
        except Exception as e:
            raise EntropyCalculationError(f"Entropy modifier calculation failed: {e}")
    
    def _calculate_harmonic_pressure(self, base_frequency: float, tiu: float) -> float:
        """
        Calculate harmonic pressure from base frequency and TIU.
        
        This implements the inversion pressure calculation from PHP.
        
        Args:
            base_frequency: Base harmonic frequency
            tiu: TIU value
            
        Returns:
            Harmonic pressure value
        """
        try:
            if base_frequency <= 0:
                raise HarmonicFrequencyError("Base frequency must be positive")
            
            # Calculate pressure using harmonic principles
            pressure = 1.0 / base_frequency
            pressure *= math.log10(abs(tiu) + 1)  # TIU influence
            pressure = pressure % 1.0  # Normalize to [0, 1)
            
            return round(pressure, self.precision)
            
        except Exception as e:
            raise HarmonicFrequencyError(f"Harmonic pressure calculation failed: {e}")
    
    def _perform_hash_rounds(self, data: bytes, rounds: int) -> str:
        """
        Perform multiple rounds of SHA256 hashing.
        
        This matches the PHP implementation's round logic.
        
        Args:
            data: Input data as bytes
            rounds: Number of hash rounds
            
        Returns:
            Final hash as hex string
        """
        if rounds < 1:
            raise CodexHashError("Rounds must be at least 1")
        
        # Initial hash
        result = hashlib.sha256(data).digest()
        
        # Additional rounds
        for _ in range(1, rounds):
            result = hashlib.sha256(result).digest()
        
        return result.hex()
    
    def hash(
        self, 
        input_data: str, 
        salt: Optional[str] = None,
        tiu: Optional[float] = None,
        rounds: int = DEFAULT_ROUNDS,
        mode: TestMode = TestMode.NON_DETERMINISTIC
    ) -> CodexHashResult:
        """
        Hash input data with full CodexHash algorithm.
        
        Args:
            input_data: The string to hash
            salt: Optional salt (None for auto-generation)
            tiu: Optional TIU value (None for auto-calculation)
            rounds: Number of hash rounds
            mode: Test execution mode
            
        Returns:
            CodexHashResult with complete metadata
        """
        start_time_ns = time.perf_counter_ns()
        
        try:
            # Set test mode
            self.time_provider.set_test_mode(mode)
            
            # Generate or validate salt
            if salt is None:
                salt = self._generate_salt()
            
            # Calculate or validate TIU
            if tiu is None:
                if mode == TestMode.DETERMINISTIC:
                    tiu = self.time_provider.get_deterministic_tiu()
                else:
                    tiu = self.time_provider.get_current_tiu()
            else:
                if not self.time_provider.validate_tiu(tiu):
                    raise InvalidTIUError(tiu, "Provided TIU value is invalid")
            
            # Get base frequency for calculations
            base_frequency = self.get_base_frequency()
            
            # Calculate entropy modifier
            entropy_modifier = self._calculate_entropy_modifier(input_data, salt, tiu)
            
            # Calculate harmonic pressure
            harmonic_pressure = self._calculate_harmonic_pressure(base_frequency, tiu)
            
            # Prepare data for hashing
            tiu_str = f"{tiu:.{self.precision}f}"
            combined_data = f"{input_data}{salt}{tiu_str}{entropy_modifier:.{self.precision}f}"
            
            # Convert to bytes
            data_bytes = combined_data.encode('utf-8')
            
            # Perform hash rounds
            final_hash = self._perform_hash_rounds(data_bytes, rounds)
            
            # Calculate execution time
            end_time_ns = time.perf_counter_ns()
            execution_time_ns = end_time_ns - start_time_ns
            
            # Calculate salt entropy for metadata
            salt_entropy = len(set(salt)) / len(salt) if salt else 0.0
            
            # Update statistics
            self._stats['total_hashes'] += 1
            self._stats['total_time_ns'] += execution_time_ns
            self._stats['avg_time_ns'] = self._stats['total_time_ns'] / self._stats['total_hashes']
            
            # Cache last values
            self._last_salt = salt
            self._last_tiu = tiu
            
            # Prepare calculation metadata
            calculation_metadata = {
                'mode': mode.value,
                'base_frequency': base_frequency,
                'combined_data_length': len(combined_data),
                'data_bytes_length': len(data_bytes),
                'tiu_string': tiu_str,
                'harmonic_calculations': {
                    'entropy_modifier': entropy_modifier,
                    'harmonic_pressure': harmonic_pressure,
                    'base_frequency': base_frequency
                },
                'statistics': dict(self._stats)
            }
            
            return CodexHashResult(
                hash_value=final_hash,
                salt=salt,
                tiu=tiu,
                rounds=rounds,
                execution_time_ns=execution_time_ns,
                base_frequency=base_frequency,
                entropy_score=entropy_modifier,
                metadata={'mode': mode.value, 'precision': self.precision},
                entropy_modifier=entropy_modifier,
                harmonic_pressure=harmonic_pressure,
                salt_entropy=salt_entropy,
                calculation_metadata=calculation_metadata
            )
            
        except Exception as e:
            if isinstance(e, (CodexHashError, InvalidTIUError)):
                raise
            else:
                raise CodexHashError(f"Hash operation failed: {e}") from e
    
    def verify(
        self,
        input_data: str,
        expected_hash: str,
        salt: str,
        tiu: float,
        rounds: int,
        buffer: float = DEFAULT_BUFFER
    ) -> bool:
        """
        Verify hash with TIU buffer for time-based variations.
        
        Args:
            input_data: Original input data
            expected_hash: Expected hash value
            salt: Salt used in original hash
            tiu: TIU value used in original hash
            rounds: Number of rounds used
            buffer: Buffer for TIU variations
            
        Returns:
            True if hash matches within buffer, False otherwise
        """
        try:
            # Generate TIU candidates within buffer
            tiu_candidates = self.time_provider.generate_buffer_candidates(tiu, buffer)
            
            # Try each TIU candidate
            for candidate_tiu in tiu_candidates:
                result = self.hash(
                    input_data, 
                    salt=salt, 
                    tiu=candidate_tiu, 
                    rounds=rounds,
                    mode=TestMode.DETERMINISTIC
                )
                
                if result.hash_value == expected_hash:
                    return True
            
            return False
            
        except Exception as e:
            raise HashValidationError(
                expected=expected_hash,
                actual="verification_failed",
                context=f"Verification error: {e}"
            )
    
    def benchmark_hash(
        self,
        input_data: str,
        iterations: int = 1000,
        mode: TestMode = TestMode.PERFORMANCE
    ) -> Dict[str, Any]:
        """
        Perform benchmark testing with detailed metrics.
        
        Args:
            input_data: Data to hash for benchmarking
            iterations: Number of iterations
            mode: Test mode for benchmarking
            
        Returns:
            Dictionary with performance metrics
        """
        times = []
        hash_results = []
        
        # Warm-up iteration
        self.hash(input_data, mode=mode)
        
        # Benchmark iterations
        for _ in range(iterations):
            start_time = time.perf_counter_ns()
            result = self.hash(input_data, mode=mode)
            end_time = time.perf_counter_ns()
            
            execution_time = end_time - start_time
            times.append(execution_time)
            hash_results.append(result)
        
        # Calculate statistics
        avg_time = sum(times) / len(times)
        min_time = min(times)
        max_time = max(times)
        
        # Calculate standard deviation
        variance = sum((t - avg_time) ** 2 for t in times) / len(times)
        std_dev = math.sqrt(variance)
        
        # Operations per second
        ops_per_sec = 1_000_000_000 / avg_time if avg_time > 0 else 0
        
        return {
            'iterations': iterations,
            'avg_time_ns': round(avg_time, 2),
            'min_time_ns': min_time,
            'max_time_ns': max_time,
            'std_dev_ns': round(std_dev, 2),
            'ops_per_sec': round(ops_per_sec, 2),
            'mode': mode.value,
            'input_length': len(input_data),
            'total_time_ns': sum(times),
            'hash_sample': hash_results[0].hash_value if hash_results else None
        }
    
    def stress_test(
        self,
        input_sizes: List[int],
        rounds_range: Tuple[int, int] = (100, 10000)
    ) -> Dict[str, Any]:
        """
        Perform stress testing with varying input sizes and rounds.
        
        Args:
            input_sizes: List of input sizes to test
            rounds_range: Tuple of (min_rounds, max_rounds)
            
        Returns:
            Dictionary with stress test results
        """
        results = {}
        min_rounds, max_rounds = rounds_range
        
        for size in input_sizes:
            # Generate test input of specified size
            test_input = 'A' * size
            
            # Test different round counts
            size_results = {}
            for rounds in [min_rounds, (min_rounds + max_rounds) // 2, max_rounds]:
                benchmark = self.benchmark_hash(
                    test_input, 
                    iterations=100,  # Fewer iterations for stress test
                    mode=TestMode.PERFORMANCE
                )
                
                # Add rounds info
                benchmark['rounds'] = rounds
                size_results[f'rounds_{rounds}'] = benchmark
            
            results[f'size_{size}'] = size_results
        
        return {
            'stress_test_results': results,
            'input_sizes': input_sizes,
            'rounds_range': rounds_range,
            'test_mode': TestMode.PERFORMANCE.value
        }
    
    def get_last_salt(self) -> Optional[str]:
        """Get the last salt used in hashing."""
        return self._last_salt
    
    def get_last_tiu(self) -> Optional[float]:
        """Get the last TIU used in hashing."""
        return self._last_tiu
    
    def get_statistics(self) -> Dict[str, Any]:
        """Get hashing statistics."""
        return dict(self._stats)
    
    def reset_statistics(self) -> None:
        """Reset hashing statistics."""
        self._stats = {
            'total_hashes': 0,
            'total_time_ns': 0,
            'avg_time_ns': 0.0
        }
    
    def generate_test_data(self, config: Dict[str, Any]) -> str:
        """
        Generate hash from input data (static method equivalent).
        
        Args:
            config: Dictionary with input data
            
        Returns:
            Generated hash value
        """
        # Convert config to JSON-like string
        import json
        data_string = json.dumps(config, separators=(',', ':'), sort_keys=True)
        
        # Hash with default parameters
        result = self.hash(data_string, rounds=DEFAULT_ROUNDS, mode=TestMode.DETERMINISTIC)
        return result.hash_value


# Utility functions for backward compatibility and testing

def generate_hash(data: Dict[str, Any]) -> str:
    """
    Generate hash from data dictionary (matches PHP static method).
    
    Args:
        data: Input data dictionary
        
    Returns:
        Generated hash string
    """
    hasher = CodexHarmonicHash()
    return hasher.generate_test_data(data)


def calculate_drift(expected_hash: str, actual_hash: str) -> float:
    """
    Calculate drift between two hashes (matches PHP static method).
    
    Args:
        expected_hash: Expected hash value
        actual_hash: Actual hash value
        
    Returns:
        Drift percentage as float
    """
    if not expected_hash or not actual_hash:
        return 1.0  # Maximum drift
    
    # Simple character-by-character comparison
    matches = sum(1 for a, b in zip(expected_hash, actual_hash) if a == b)
    max_length = max(len(expected_hash), len(actual_hash))
    
    if max_length == 0:
        return 0.0
    
    similarity = matches / max_length
    drift = 1.0 - similarity
    
    return round(drift, 6)
