"""
CodexHash Time Utilities
========================

Time and TIU (Time Integrity Unit) calculation utilities designed for:
1. Deterministic testing with fixed TIU values
2. Performance benchmarking with optimized calculations
3. Cryptographic analysis with precise timing
4. Robustness testing with edge cases
"""

import time
import math
import random
import requests
import json
from typing import Optional, List, Dict, Any
from dataclasses import dataclass

from .interfaces import AbstractTimeProvider, TestMode
from .exceptions import InvalidTIUError, TimeProviderError


# Physical constants for harmonic calculations (from PHP implementation)
C = 299792458.0              # Speed of light in m/s
AU = 149597870700.0          # Astronomical unit in meters  
PLANCK_TIME = 5.391247e-44   # Planck time in seconds
PLANCK_FREQ = 1.0 / PLANCK_TIME  # Planck frequency in Hz

# TIU validation bounds - Keep within reasonable range for hash algorithms
MIN_TIU = 0.0
MAX_TIU = 10.0  # Reasonable upper bound for TIU values


@dataclass
class TIUCalculationResult:
    """Result of TIU calculation with metadata."""
    tiu: float
    base_frequency: float
    calculation_method: str
    timestamp: float
    precision: int
    metadata: Dict[str, Any]


class TIUCalculator:
    """
    Advanced TIU calculator supporting multiple calculation modes
    for comprehensive testing and validation.
    """
    
    def __init__(self, precision: int = 6):
        """
        Initialize TIU calculator.
        
        Args:
            precision: Decimal places for TIU calculations
        """
        self.precision = precision
        self._deterministic_seed = None
        self._base_frequency_cache = None
        
    def calculate_base_frequency(self) -> float:
        """
        Calculate base harmonic frequency using physical constants.
        
        This matches the PHP implementation:
        fBase = sqrt(C / (AU * PLANCK_TIME))
        """
        if self._base_frequency_cache is None:
            try:
                self._base_frequency_cache = math.sqrt(C / (AU * PLANCK_TIME))
            except (ValueError, ZeroDivisionError) as e:
                raise TimeProviderError(f"Base frequency calculation failed: {e}")
        
        return self._base_frequency_cache
    
    def calculate_current_tiu(self) -> TIUCalculationResult:
        """
        Calculate TIU based on current system time.
        
        Returns:
            TIUCalculationResult with current TIU and metadata
        """
        timestamp = time.time()
        
        try:
            # Use harmonic physics to derive reasonable TIU from timestamp
            # Take fractional part and scale appropriately
            fractional_time = timestamp % 1.0
            base_freq = self.calculate_base_frequency()
            
            # Generate TIU within reasonable bounds using harmonic oscillation
            harmonic_component = math.sin(fractional_time * 2 * math.pi)
            tiu = 1.0 + (harmonic_component * 2.0)  # Range: [-1, 3], mostly [0, 2]
            tiu = max(MIN_TIU, min(MAX_TIU, tiu))  # Ensure within bounds
            tiu = round(tiu, self.precision)
            
            if not self.validate_tiu(tiu):
                raise InvalidTIUError(tiu, "TIU outside valid range")
            
            return TIUCalculationResult(
                tiu=tiu,
                base_frequency=base_freq,
                calculation_method="current_time",
                timestamp=timestamp,
                precision=self.precision,
                metadata={
                    'system_time': timestamp,
                    'fractional_time': fractional_time,
                    'harmonic_component': harmonic_component,
                    'constants': {'C': C, 'AU': AU, 'PLANCK_TIME': PLANCK_TIME}
                }
            )
            
        except Exception as e:
            raise TimeProviderError(f"Current TIU calculation failed: {e}")
    
    def calculate_deterministic_tiu(self, seed: Optional[int] = None) -> TIUCalculationResult:
        """
        Calculate deterministic TIU for reproducible testing.
        
        Args:
            seed: Random seed for deterministic generation
            
        Returns:
            TIUCalculationResult with deterministic TIU
        """
        if seed is not None:
            self._deterministic_seed = seed
        elif self._deterministic_seed is None:
            self._deterministic_seed = 12345  # Default test seed
        
        try:
            # Use seed to generate deterministic TIU
            random.seed(self._deterministic_seed)
            
            # Generate TIU in reasonable range for testing
            tiu = random.uniform(0.1, 5.0)  # Keep within reasonable bounds
            tiu = round(tiu, self.precision)
            
            base_freq = self.calculate_base_frequency()
            
            return TIUCalculationResult(
                tiu=tiu,
                base_frequency=base_freq,
                calculation_method="deterministic",
                timestamp=time.time(),
                precision=self.precision,
                metadata={
                    'seed': self._deterministic_seed,
                    'range': [1000000.0, 2000000.0],
                    'deterministic': True
                }
            )
            
        except Exception as e:
            raise TimeProviderError(f"Deterministic TIU calculation failed: {e}")
    
    def calculate_fixed_tiu(self, value: float) -> TIUCalculationResult:
        """
        Create TIU result for fixed value (used in KAT).
        
        Args:
            value: Fixed TIU value
            
        Returns:
            TIUCalculationResult with fixed TIU
        """
        if not self.validate_tiu(value):
            raise InvalidTIUError(value, "Fixed TIU value outside valid range")
        
        try:
            base_freq = self.calculate_base_frequency()
            
            return TIUCalculationResult(
                tiu=round(value, self.precision),
                base_frequency=base_freq,
                calculation_method="fixed",
                timestamp=time.time(),
                precision=self.precision,
                metadata={
                    'fixed_value': value,
                    'validation_passed': True
                }
            )
            
        except Exception as e:
            raise TimeProviderError(f"Fixed TIU calculation failed: {e}")
    
    def calculate_tiu_from_timestamp(self, timestamp: float) -> TIUCalculationResult:
        """
        Calculate TIU from a given timestamp (from time server).
        
        Args:
            timestamp: Unix timestamp from time server
            
        Returns:
            TIUCalculationResult with calculated TIU
        """
        try:
            # Calculate TIU using the server timestamp
            base_freq = self.calculate_base_frequency()
            
            # Use harmonic physics to derive TIU from timestamp
            # This mimics the PHP implementation's approach
            planck_time = 5.391247e-44
            c_speed = 299792458  # Speed of light
            
            # Calculate temporal distortion factor
            temporal_factor = (timestamp % 1.0) * c_speed
            harmonic_distortion = math.sin(temporal_factor * planck_time * base_freq)
            
            # Generate TIU value within valid bounds
            tiu_raw = 0.5 + (harmonic_distortion * 0.3)  # Keep within reasonable bounds
            tiu = max(MIN_TIU, min(MAX_TIU, tiu_raw))
            
            return TIUCalculationResult(
                tiu=round(tiu, self.precision),
                base_frequency=base_freq,
                calculation_method="server_timestamp",
                timestamp=timestamp,
                precision=self.precision,
                metadata={
                    'server_timestamp': timestamp,
                    'temporal_factor': temporal_factor,
                    'harmonic_distortion': harmonic_distortion
                }
            )
            
        except Exception as e:
            raise TimeProviderError(f"Server timestamp TIU calculation failed: {e}")
            
        except Exception as e:
            raise TimeProviderError(f"Fixed TIU calculation failed: {e}")
    
    def generate_buffer_candidates(self, tiu: float, buffer: float = 0.000001) -> List[float]:
        """
        Generate TIU candidates within buffer range for verification.
        
        This matches the PHP implementation's buffer logic for hash verification.
        
        Args:
            tiu: Base TIU value
            buffer: Buffer range for variations
            
        Returns:
            List of TIU candidates to try
        """
        if not self.validate_tiu(tiu):
            raise InvalidTIUError(tiu, "Base TIU value invalid for buffer generation")
        
        candidates = [tiu]  # Always include exact value first
        
        # Generate candidates within buffer range
        for i in range(1, 6):  # Generate 5 candidates in each direction
            positive_candidate = tiu + (buffer * i)
            negative_candidate = tiu - (buffer * i)
            
            if self.validate_tiu(positive_candidate):
                candidates.append(round(positive_candidate, self.precision))
            
            if self.validate_tiu(negative_candidate):
                candidates.append(round(negative_candidate, self.precision))
        
        return list(set(candidates))  # Remove duplicates
    
    def validate_tiu(self, tiu: float) -> bool:
        """
        Validate TIU value is within acceptable range.
        
        Args:
            tiu: TIU value to validate
            
        Returns:
            True if valid, False otherwise
        """
        return (
            isinstance(tiu, (int, float)) and
            not math.isnan(tiu) and
            not math.isinf(tiu) and
            MIN_TIU <= tiu <= MAX_TIU
        )


class CodexTime(AbstractTimeProvider):
    """
    Main time provider implementation supporting all test modes.
    
    This class provides the interface used by CodexHarmonicHash
    and supports comprehensive testing scenarios.
    """
    
    def __init__(self, precision: int = 6, server_url: str = "http://localhost:8000", fallback_mode: bool = True):
        """
        Initialize CodexTime provider.
        
        Args:
            precision: Decimal places for TIU calculations
            server_url: URL of the CodexTime server
            fallback_mode: Whether to fallback to local time if server unavailable
        """
        self.calculator = TIUCalculator(precision)
        self.server_url = server_url
        self.fallback_mode = fallback_mode
        self._current_mode = TestMode.NON_DETERMINISTIC
        self._deterministic_seed = None
        self._server_available = self._test_connection()
    
    def _test_connection(self) -> bool:
        """Test connection to CodexTime server."""
        try:
            response = requests.get(f"{self.server_url}/health", timeout=5)
            if response.status_code == 200:
                return True
            else:
                if not self.fallback_mode:
                    raise TimeProviderError(f"CodexTime server not healthy: {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            if not self.fallback_mode:
                raise TimeProviderError(f"Cannot connect to CodexTime server at {self.server_url}: {e}")
            return False
    
    def _test_connection(self) -> None:
        """Test connection to CodexTime server."""
        try:
            response = requests.get(f"{self.server_url}/health", timeout=5)
            if response.status_code != 200:
                raise TimeProviderError(f"CodexTime server not healthy: {response.status_code}")
        except requests.exceptions.RequestException as e:
            raise TimeProviderError(f"Cannot connect to CodexTime server at {self.server_url}: {e}")
    
    def _get_server_time(self) -> Dict[str, Any]:
        """Get time data from CodexTime server."""
        try:
            response = requests.get(f"{self.server_url}/time", timeout=5)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            raise TimeProviderError(f"Failed to get time from server: {e}")
    
    def set_test_mode(self, mode: TestMode, seed: Optional[int] = None) -> None:
        """
        Set test execution mode.
        
        Args:
            mode: Test mode to use
            seed: Seed for deterministic modes
        """
        self._current_mode = mode
        if seed is not None:
            self._deterministic_seed = seed
    
    def get_current_tiu(self) -> float:
        """
        Get current TIU based on current test mode.
        
        Returns:
            TIU value appropriate for current mode
        """
        if self._current_mode == TestMode.DETERMINISTIC:
            result = self.calculator.calculate_deterministic_tiu(self._deterministic_seed)
        elif self._current_mode == TestMode.NON_DETERMINISTIC and self._server_available:
            # Get real time from server
            time_data = self._get_server_time()
            # Use server timestamp for TIU calculation
            timestamp = time_data.get('timestamp', time.time())
            result = self.calculator.calculate_tiu_from_timestamp(timestamp)
        else:
            # Fallback to calculated current time
            result = self.calculator.calculate_current_tiu()
        
        return result.tiu
    
    def get_deterministic_tiu(self, seed: Optional[int] = None) -> float:
        """
        Get deterministic TIU for reproducible testing.
        
        Args:
            seed: Random seed for generation
            
        Returns:
            Deterministic TIU value
        """
        result = self.calculator.calculate_deterministic_tiu(seed)
        return result.tiu
    
    def get_fixed_tiu(self, value: float) -> float:
        """
        Get validated fixed TIU value.
        
        Args:
            value: Fixed TIU value
            
        Returns:
            Validated TIU value
        """
        result = self.calculator.calculate_fixed_tiu(value)
        return result.tiu
    
    def validate_tiu(self, tiu: float) -> bool:
        """
        Validate TIU value.
        
        Args:
            tiu: TIU value to validate
            
        Returns:
            True if valid, False otherwise
        """
        return self.calculator.validate_tiu(tiu)
    
    def generate_buffer_candidates(self, tiu: float, buffer: float = 0.000001) -> List[float]:
        """
        Generate TIU buffer candidates for verification.
        
        Args:
            tiu: Base TIU value
            buffer: Buffer range
            
        Returns:
            List of TIU candidates
        """
        return self.calculator.generate_buffer_candidates(tiu, buffer)
    
    def get_base_frequency(self) -> float:
        """
        Get harmonic base frequency.
        
        Returns:
            Base frequency value
        """
        return self.calculator.calculate_base_frequency()
    
    def get_calculation_metadata(self, tiu: float) -> Dict[str, Any]:
        """
        Get metadata for a TIU calculation.
        
        Args:
            tiu: TIU value to get metadata for
            
        Returns:
            Dictionary with calculation metadata
        """
        return {
            'tiu': tiu,
            'base_frequency': self.get_base_frequency(),
            'mode': self._current_mode.value,
            'seed': self._deterministic_seed,
            'precision': self.calculator.precision,
            'valid': self.validate_tiu(tiu),
            'constants': {
                'C': C,
                'AU': AU, 
                'PLANCK_TIME': PLANCK_TIME,
                'PLANCK_FREQ': PLANCK_FREQ
            }
        }
