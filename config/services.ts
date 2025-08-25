// Services configuration for dynamic menu generation
export interface ServiceConfig {
    name: string;
    href: string;
    description?: string;
    icon?: string;
}

export const services: Record<string, ServiceConfig> = {
    QuantumHash: {
        name: "Quantum Hash",
        href: "/services/quantum-hash",
        description: "Quantum-resistant hashing algorithms",
        icon: "ph ph-shield-plus"
    },
    APIService: {
        name: "Hash API",
        href: "/services/api",
        description: "RESTful hashing API endpoints",
        icon: "ph ph-code"
    },
    Analytics: {
        name: "Analytics",
        href: "/services/analytics",
        description: "Hash performance and security analytics",
        icon: "ph ph-chart-line"
    },
    Enterprise: {
        name: "Enterprise",
        href: "/services/enterprise",
        description: "Enterprise-grade hashing solutions",
        icon: "ph ph-buildings"
    },
    Developer: {
        name: "Developer Tools",
        href: "/services/developer",
        description: "SDKs, libraries, and development tools",
        icon: "ph ph-terminal-window"
    }
};

export default services;