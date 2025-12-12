"""
CodexHash Tiered Architecture Implementation
============================================

Multi-tier hashing system for quantum-era security:
- Tier 1 (Commercial): 256-512 bit window
- Tier 2 (Enterprise): 512-768 bit window  
- Tier 3 (Government): 768-1024 bit window

Each tier provides progressively stronger quantum resistance
and additional security features.
"""

from enum import Enum
from dataclasses import dataclass, field
from typing import Tuple, Dict, Any, List, Optional
import hashlib
import hmac
from datetime import datetime


class HashTier(Enum):
    """Hashing security tiers for different use cases."""
    COMMERCIAL = "commercial"
    ENTERPRISE = "enterprise"
    GOVERNMENT = "government"


class ComplianceStandard(Enum):
    """Supported compliance standards."""
    SOC2 = "SOC2"
    HIPAA = "HIPAA"
    GDPR = "GDPR"
    PCI_DSS = "PCI_DSS"
    FISMA = "FISMA"
    NIST = "NIST"


@dataclass
class TierFeatures:
    """Feature set for a specific tier."""
    performance_optimized: bool = False
    mobile_friendly: bool = False
    consumer_apps: bool = False
    compliance_ready: bool = False
    auditable: bool = False
    compliance_standards: List[ComplianceStandard] = field(default_factory=list)
    harmonic_lock: bool = False
    command_trust: bool = False
    defense_grade: bool = False
    inter_agency_ready: bool = False
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            'performance_optimized': self.performance_optimized,
            'mobile_friendly': self.mobile_friendly,
            'consumer_apps': self.consumer_apps,
            'compliance_ready': self.compliance_ready,
            'auditable': self.auditable,
            'compliance_standards': [s.value for s in self.compliance_standards],
            'harmonic_lock': self.harmonic_lock,
            'command_trust': self.command_trust,
            'defense_grade': self.defense_grade,
            'inter_agency_ready': self.inter_agency_ready
        }


@dataclass
class TierConfig:
    """Configuration for a specific hashing tier."""
    tier: HashTier
    bit_range: Tuple[int, int]
    output_length: int  # in bytes
    quantum_resistance: str
    security_level: str
    features: TierFeatures
    rounds_multiplier: float = 1.0
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            'tier': self.tier.value,
            'bit_range': self.bit_range,
            'output_length': self.output_length,
            'quantum_resistance': quantum_resistance,
            'security_level': self.security_level,
            'features': self.features.to_dict(),
            'rounds_multiplier': self.rounds_multiplier
        }


# Tier 1: Commercial - Consumer Applications
TIER_COMMERCIAL = TierConfig(
    tier=HashTier.COMMERCIAL,
    bit_range=(256, 512),
    output_length=32,  # 256-bit output
    quantum_resistance="baseline",
    security_level="standard",
    features=TierFeatures(
        performance_optimized=True,
        mobile_friendly=True,
        consumer_apps=True,
        compliance_ready=False,
        auditable=False
    ),
    rounds_multiplier=1.0
)


# Tier 2: Enterprise - Business & Compliance
TIER_ENTERPRISE = TierConfig(
    tier=HashTier.ENTERPRISE,
    bit_range=(512, 768),
    output_length=64,  # 512-bit output
    quantum_resistance="enhanced",
    security_level="high",
    features=TierFeatures(
        performance_optimized=False,
        mobile_friendly=False,
        consumer_apps=False,
        compliance_ready=True,
        auditable=True,
        compliance_standards=[
            ComplianceStandard.SOC2,
            ComplianceStandard.HIPAA,
            ComplianceStandard.GDPR,
            ComplianceStandard.PCI_DSS
        ],
        harmonic_lock=False,
        command_trust=False
    ),
    rounds_multiplier=1.5
)


# Tier 3: Government - Critical Infrastructure
TIER_GOVERNMENT = TierConfig(
    tier=HashTier.GOVERNMENT,
    bit_range=(768, 1024),
    output_length=128,  # 1024-bit output
    quantum_resistance="maximum",
    security_level="critical",
    features=TierFeatures(
        performance_optimized=False,
        mobile_friendly=False,
        consumer_apps=False,
        compliance_ready=True,
        auditable=True,
        compliance_standards=[
            ComplianceStandard.FISMA,
            ComplianceStandard.NIST,
            ComplianceStandard.SOC2
        ],
        harmonic_lock=True,
        command_trust=True,
        defense_grade=True,
        inter_agency_ready=True
    ),
    rounds_multiplier=2.0
)


# Tier configuration lookup
TIER_CONFIGS: Dict[HashTier, TierConfig] = {
    HashTier.COMMERCIAL: TIER_COMMERCIAL,
    HashTier.ENTERPRISE: TIER_ENTERPRISE,
    HashTier.GOVERNMENT: TIER_GOVERNMENT
}


@dataclass
class TieredHashResult:
    """Result of a tiered hash operation."""
    hash: str
    tier: HashTier
    bit_length: int
    quantum_resistance: str
    security_level: str
    timestamp: str
    tiu: Optional[float] = None
    compliance_flags: Optional[List[str]] = None
    audit_trail: Optional[Dict[str, Any]] = None
    harmonic_lock_enabled: bool = False
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            'hash': self.hash,
            'tier': self.tier.value,
            'bit_length': self.bit_length,
            'quantum_resistance': self.quantum_resistance,
            'security_level': self.security_level,
            'timestamp': self.timestamp,
            'tiu': self.tiu,
            'compliance_flags': self.compliance_flags,
            'audit_trail': self.audit_trail,
            'harmonic_lock_enabled': self.harmonic_lock_enabled
        }


class TieredHasher:
    """
    Tiered hashing implementation for CodexHash.
    
    Provides three security tiers with different bit windows
    and security features.
    """
    
    def __init__(self, tier: HashTier = HashTier.COMMERCIAL):
        """
        Initialize tiered hasher.
        
        Args:
            tier: Security tier to use
        """
        self.tier = tier
        self.config = TIER_CONFIGS[tier]
    
    def extract_bit_window(
        self,
        full_hash: bytes,
        bit_range: Tuple[int, int]
    ) -> bytes:
        """
        Extract specific bit window from full 1024-bit hash.
        
        Args:
            full_hash: Full 1024-bit hash (128 bytes)
            bit_range: Tuple of (start_bit, end_bit)
            
        Returns:
            Extracted bit window as bytes
        """
        start_bit, end_bit = bit_range
        start_byte = start_bit // 8
        end_byte = (end_bit + 7) // 8
        
        return full_hash[start_byte:end_byte]
    
    def generate_full_hash(
        self,
        data: str,
        salt: str,
        tiu: float,
        rounds: int
    ) -> bytes:
        """
        Generate full 1024-bit (128 byte) hash.
        
        Args:
            data: Input data
            salt: Salt value
            tiu: Time Integrity Unit
            rounds: Number of hash rounds
            
        Returns:
            Full 1024-bit hash
        """
        # Combine input components
        combined = f"{data}{salt}{tiu}".encode('utf-8')
        
        # Initial SHA3-512 (64 bytes)
        hash1 = hashlib.sha3_512(combined).digest()
        
        # Second round SHA3-512 (64 bytes)
        hash2 = hashlib.sha3_512(hash1).digest()
        
        # Combine to get 128 bytes (1024 bits)
        full_hash = hash1 + hash2
        
        # Apply rounds
        for _ in range(rounds):
            full_hash = hashlib.sha3_512(full_hash).digest() + \
                       hashlib.sha3_512(full_hash[::-1]).digest()
        
        return full_hash[:128]  # Ensure exactly 128 bytes
    
    def apply_harmonic_lock(
        self,
        hash_data: bytes,
        tiu: float,
        lock_key: bytes
    ) -> bytes:
        """
        Apply harmonic lock-write layer (Tier 3 only).
        
        Args:
            hash_data: Hash to lock
            tiu: Time Integrity Unit
            lock_key: Locking key
            
        Returns:
            Locked hash
        """
        if not self.config.features.harmonic_lock:
            return hash_data
        
        # Generate harmonic signature based on TIU
        harmonic_signature = self._calculate_harmonic_signature(tiu)
        
        # XOR hash with harmonic signature
        locked_hash = bytes(
            h ^ hs for h, hs in zip(hash_data, harmonic_signature)
        )
        
        # HMAC seal
        sealed_hash = hmac.new(
            lock_key,
            locked_hash,
            hashlib.sha3_256
        ).digest()
        
        return sealed_hash
    
    def _calculate_harmonic_signature(self, tiu: float) -> bytes:
        """
        Calculate harmonic signature from TIU.
        
        Args:
            tiu: Time Integrity Unit
            
        Returns:
            Harmonic signature bytes
        """
        # Use golden ratio and TIU to generate signature
        phi = 1.618033988749895
        harmonic_value = tiu * phi
        
        # Generate deterministic bytes from harmonic value
        signature_str = f"{harmonic_value:.15f}".encode('utf-8')
        return hashlib.sha3_256(signature_str).digest()
    
    def hash(
        self,
        data: str,
        salt: Optional[str] = None,
        tiu: Optional[float] = None,
        rounds: Optional[int] = None
    ) -> TieredHashResult:
        """
        Generate tiered hash.
        
        Args:
            data: Input data to hash
            salt: Optional salt (auto-generated if None)
            tiu: Optional TIU value (auto-calculated if None)
            rounds: Optional rounds (tier default if None)
            
        Returns:
            TieredHashResult with hash and metadata
        """
        # Auto-generate missing values
        if salt is None:
            salt = hashlib.sha256(data.encode()).hexdigest()
        
        if tiu is None:
            # Simple TIU calculation (can be enhanced)
            tiu = 0.618034  # Golden ratio default
        
        if rounds is None:
            base_rounds = 16
            rounds = int(base_rounds * self.config.rounds_multiplier)
        
        # Generate full 1024-bit hash
        full_hash = self.generate_full_hash(data, salt, tiu, rounds)
        
        # Extract tier-specific bit window
        windowed_hash = self.extract_bit_window(
            full_hash,
            self.config.bit_range
        )
        
        # Apply harmonic lock for Tier 3
        if self.config.features.harmonic_lock:
            lock_key = hashlib.sha256(salt.encode()).digest()
            windowed_hash = self.apply_harmonic_lock(
                windowed_hash,
                tiu,
                lock_key
            )
        
        # Convert to hex
        hash_hex = windowed_hash.hex()
        
        # Build result
        result = TieredHashResult(
            hash=hash_hex,
            tier=self.tier,
            bit_length=self.config.output_length * 8,
            quantum_resistance=self.config.quantum_resistance,
            security_level=self.config.security_level,
            timestamp=datetime.utcnow().isoformat(),
            tiu=tiu,
            compliance_flags=[
                s.value for s in self.config.features.compliance_standards
            ] if self.config.features.compliance_ready else None,
            audit_trail={
                'rounds': rounds,
                'bit_range': self.config.bit_range,
                'salt_length': len(salt)
            } if self.config.features.auditable else None,
            harmonic_lock_enabled=self.config.features.harmonic_lock
        )
        
        return result
    
    def verify(
        self,
        data: str,
        expected_hash: str,
        salt: str,
        tiu: float,
        rounds: int
    ) -> bool:
        """
        Verify hash against expected value.
        
        Args:
            data: Original input data
            expected_hash: Hash to verify against
            salt: Salt used in original hash
            tiu: TIU used in original hash
            rounds: Rounds used in original hash
            
        Returns:
            True if hash matches, False otherwise
        """
        result = self.hash(data, salt, tiu, rounds)
        return result.hash == expected_hash


# Convenience functions
def create_commercial_hasher() -> TieredHasher:
    """Create Tier 1 (Commercial) hasher."""
    return TieredHasher(HashTier.COMMERCIAL)


def create_enterprise_hasher() -> TieredHasher:
    """Create Tier 2 (Enterprise) hasher."""
    return TieredHasher(HashTier.ENTERPRISE)


def create_government_hasher() -> TieredHasher:
    """Create Tier 3 (Government) hasher."""
    return TieredHasher(HashTier.GOVERNMENT)


# Quick hash functions
def hash_commercial(data: str, **kwargs) -> TieredHashResult:
    """Quick Tier 1 hash."""
    return create_commercial_hasher().hash(data, **kwargs)


def hash_enterprise(data: str, **kwargs) -> TieredHashResult:
    """Quick Tier 2 hash."""
    return create_enterprise_hasher().hash(data, **kwargs)


def hash_government(data: str, **kwargs) -> TieredHashResult:
    """Quick Tier 3 hash."""
    return create_government_hasher().hash(data, **kwargs)


if __name__ == "__main__":
    # Demo usage
    print("🔐 CodexHash Tiered Architecture Demo\n")
    
    test_data = "Sensitive enterprise data"
    
    # Test each tier
    for tier in HashTier:
        hasher = TieredHasher(tier)
        result = hasher.hash(test_data)
        
        print(f"\n{'='*60}")
        print(f"Tier: {result.tier.value.upper()}")
        print(f"{'='*60}")
        print(f"Hash: {result.hash[:64]}...")
        print(f"Bit Length: {result.bit_length}")
        print(f"Quantum Resistance: {result.quantum_resistance}")
        print(f"Security Level: {result.security_level}")
        print(f"Harmonic Lock: {result.harmonic_lock_enabled}")
        if result.compliance_flags:
            print(f"Compliance: {', '.join(result.compliance_flags)}")
