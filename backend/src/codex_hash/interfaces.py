"""
CodexHash Interface Definitions
==============================

Abstract interfaces for CodexHash components, designed to support:
1. Comprehensive testing and benchmarking
2. Multiple implementation backends
3. MCP server integration
4. Performance analysis
5. Cryptographic validation
"""

from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional, Protocol, Union, Tuple
from dataclasses import dataclass
from enum import Enum
import time


class TestMode(Enum):
    """Test execution modes for different validation scenarios."""
    DETERMINISTIC = "deterministic"     # Fixed TIU, reproducible results
    NON_DETERMINISTIC = "non_deterministic"  # Current time TIU
    PERFORMANCE = "performance"         # Optimized for speed benchmarks
    CRYPTOGRAPHIC = "cryptographic"     # Full entropy and security validation
    ROBUSTNESS = "robustness"          # Stress testing with edge cases


@dataclass
class HashResult:
    """Standardized hash result with full metadata for testing."""
    hash_value: str
    salt: str
    tiu: float
    rounds: int
    execution_time_ns: int
    base_frequency: float
    entropy_score: float
    metadata: Dict[str, Any]
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            'hash': self.hash_value,
            'salt': self.salt,
            'tiu': self.tiu,
            'rounds': self.rounds,
            'execution_time_ns': self.execution_time_ns,
            'base_frequency': self.base_frequency,
            'entropy_score': self.entropy_score,
            'metadata': self.metadata
        }


class TimeProvider(Protocol):
    """Protocol for time providers supporting different test modes."""
    
    def get_current_tiu(self) -> float:
        """Get current TIU value."""
        ...
    
    def get_deterministic_tiu(self, seed: Optional[int] = None) -> float:
        """Get deterministic TIU for reproducible testing."""
        ...
    
    def validate_tiu(self, tiu: float) -> bool:
        """Validate TIU value is within acceptable range."""
        ...


class HashFunction(Protocol):
    """Protocol for hash functions supporting comprehensive testing."""
    
    def hash(
        self, 
        input_data: str, 
        salt: Optional[str] = None,
        tiu: Optional[float] = None,
        rounds: int = 1000,
        mode: TestMode = TestMode.NON_DETERMINISTIC
    ) -> HashResult:
        """
        Hash input data with full test support.
        
        Args:
            input_data: The string to hash
            salt: Optional salt (None for auto-generation)
            tiu: Optional TIU value (None for auto-calculation)
            rounds: Number of hash rounds
            mode: Test execution mode
            
        Returns:
            HashResult with complete metadata
        """
        ...
    
    def verify(
        self,
        input_data: str,
        expected_hash: str,
        salt: str,
        tiu: float,
        rounds: int,
        buffer: float = 0.000001
    ) -> bool:
        """Verify hash with TIU buffer for time-based variations."""
        ...
    
    def get_base_frequency(self) -> float:
        """Get current base frequency for harmonic calculations."""
        ...


class BenchmarkableHash(HashFunction, Protocol):
    """Extended interface for performance benchmarking."""
    
    def benchmark_hash(
        self,
        input_data: str,
        iterations: int = 1000,
        mode: TestMode = TestMode.PERFORMANCE
    ) -> Dict[str, Any]:
        """
        Perform benchmark testing with detailed metrics.
        
        Returns:
            Dictionary with performance metrics:
            - avg_time_ns: Average execution time
            - min_time_ns: Minimum execution time
            - max_time_ns: Maximum execution time
            - std_dev_ns: Standard deviation
            - ops_per_sec: Operations per second
            - memory_usage: Peak memory usage
        """
        ...
    
    def stress_test(
        self,
        input_sizes: List[int],
        rounds_range: Tuple[int, int] = (100, 10000)
    ) -> Dict[str, Any]:
        """Perform stress testing with varying input sizes and rounds."""
        ...


class CryptographicAnalyzer(Protocol):
    """Protocol for cryptographic analysis of hash functions."""
    
    def analyze_distribution(
        self,
        hash_function: HashFunction,
        sample_size: int = 10000
    ) -> Dict[str, float]:
        """Analyze hash output distribution for uniformity."""
        ...
    
    def collision_test(
        self,
        hash_function: HashFunction,
        test_size: int = 100000
    ) -> Dict[str, Any]:
        """Test for hash collisions."""
        ...
    
    def entropy_analysis(
        self,
        hash_function: HashFunction,
        input_patterns: List[str]
    ) -> Dict[str, float]:
        """Analyze entropy characteristics."""
        ...


class KnownAnswerTestProvider(Protocol):
    """Protocol for Known Answer Test (KAT) management."""
    
    def generate_test_vectors(
        self,
        hash_function: HashFunction,
        count: int = 16
    ) -> List[Dict[str, Any]]:
        """Generate KAT vectors for validation."""
        ...
    
    def validate_vectors(
        self,
        hash_function: HashFunction,
        vectors: List[Dict[str, Any]]
    ) -> Tuple[int, int, List[str]]:
        """
        Validate test vectors.
        
        Returns:
            (passed_count, total_count, error_messages)
        """
        ...
    
    def load_vectors(self, file_path: str) -> List[Dict[str, Any]]:
        """Load test vectors from file."""
        ...
    
    def save_vectors(self, vectors: List[Dict[str, Any]], file_path: str) -> None:
        """Save test vectors to file."""
        ...


class MCPBridge(Protocol):
    """Protocol for MCP server bridge integration."""
    
    def call_remote_hash(
        self,
        input_data: str,
        salt: Optional[str] = None,
        tiu: Optional[float] = None,
        rounds: int = 1000
    ) -> HashResult:
        """Call remote hash service via MCP."""
        ...
    
    def health_check(self) -> Dict[str, Any]:
        """Check MCP service health."""
        ...
    
    def get_service_info(self) -> Dict[str, Any]:
        """Get remote service information."""
        ...


# Abstract base classes for concrete implementations

class AbstractHashFunction(ABC):
    """Abstract base class for hash function implementations."""
    
    @abstractmethod
    def hash(
        self, 
        input_data: str, 
        salt: Optional[str] = None,
        tiu: Optional[float] = None,
        rounds: int = 1000,
        mode: TestMode = TestMode.NON_DETERMINISTIC
    ) -> HashResult:
        """Hash implementation must be provided by subclasses."""
        pass
    
    @abstractmethod
    def verify(
        self,
        input_data: str,
        expected_hash: str,
        salt: str,
        tiu: float,
        rounds: int,
        buffer: float = 0.000001
    ) -> bool:
        """Verification implementation must be provided by subclasses."""
        pass
    
    @abstractmethod
    def get_base_frequency(self) -> float:
        """Base frequency calculation must be provided by subclasses."""
        pass


class AbstractTimeProvider(ABC):
    """Abstract base class for time providers."""
    
    @abstractmethod
    def get_current_tiu(self) -> float:
        """Current TIU calculation must be provided by subclasses."""
        pass
    
    @abstractmethod
    def get_deterministic_tiu(self, seed: Optional[int] = None) -> float:
        """Deterministic TIU calculation must be provided by subclasses."""
        pass
    
    @abstractmethod
    def validate_tiu(self, tiu: float) -> bool:
        """TIU validation must be provided by subclasses."""
        pass

from abc import ABC, abstractmethod
from typing import Optional, Dict, Any
from dataclasses import dataclass
import time


@dataclass
class CodexHashResult:
    """Result of a CodexHash operation."""
    hash: str
    salt: str
    tiu: float
    rounds: int
    base_frequency: float
    entropy_modifier: float
    execution_time_ms: float


class HashFunction(ABC):
    """Abstract interface for hash functions."""
    
    @abstractmethod
    def hash(self, 
             input_data: str, 
             salt: Optional[str] = None,
             tiu: Optional[float] = None,
             rounds: int = 1000) -> CodexHashResult:
        """
        Generate a hash from input data.
        
        Args:
            input_data: The data to hash
            salt: Optional salt (random if None)
            tiu: Optional Time Integrity Unit (current time if None)  
            rounds: Number of hash rounds
            
        Returns:
            CodexHashResult with hash and metadata
        """
        pass
    
    @abstractmethod
    def verify(self, 
               input_data: str,
               expected_hash: str,
               salt: str,
               tiu: float,
               rounds: int,
               buffer: float = 0.000001) -> bool:
        """
        Verify a hash against expected result.
        
        Args:
            input_data: Original input data
            expected_hash: Expected hash value
            salt: Salt used in original hash
            tiu: TIU used in original hash
            rounds: Rounds used in original hash
            buffer: TIU comparison buffer
            
        Returns:
            True if hash matches, False otherwise
        """
        pass


class TimeProvider(ABC):
    """Abstract interface for time-related calculations."""
    
    @abstractmethod
    def get_current_tiu(self) -> float:
        """Get current Time Integrity Unit."""
        pass
    
    @abstractmethod
    def generate_buffer_candidates(self, tiu: float, buffer: float) -> list[float]:
        """Generate TIU candidates within buffer range."""
        pass
    
    @abstractmethod
    def get_base_frequency(self) -> float:
        """Calculate base harmonic frequency."""
        pass
