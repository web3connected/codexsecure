"use client";

import React from "react";
import { GlobalHeader } from "../codex/panels";
import { ApplicationLogo, Navigation, DateGreeter } from "../codex/widgets";

/**
 * CodexSecureHeader - CodexSecure themed header implementation
 *
 * Local wrapper that loads the global GlobalHeader from codex components
 * with CodexSecure-specific configuration:
 * - ApplicationLogo widget (CodexSecure branding)
 * - Navigation widget (links to zones, docs, pricing)
 * - DateGreeter widget (time-based greeting)
 */
export default function CodexSecureHeader() {
  const navLinks = [
    { label: "Home", href: "/" },
    { label: "Security Zones", href: "/zones" },
    { label: "Docs", href: "/docs" },
    { label: "Pricing", href: "/pricing" },
    { label: "Support", href: "/support" },
  ];

  return (
    <GlobalHeader
      topWidget1={<DateGreeter />}
      topWidget2={null}
      topWidget3={null}
      topWidget4={null}
      bottomWidget1={<ApplicationLogo logo="CodexSecure" showIcon={true} />}
      bottomWidget2={<Navigation items={navLinks} orientation="horizontal" />}
    />
  );
}
