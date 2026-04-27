"use client";

import { ApplicationLogo } from "../../codex/widgets";

export interface PageLoaderProps {
  message?: string;
}

export default function PageLoader({
  message = "Loading...",
}: PageLoaderProps) {
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-background/80 backdrop-blur-sm z-50">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-primary"></div>
        <ApplicationLogo logo="CodexSecure" showIcon={true} />
        <p className="text-sm text-muted-foreground">{message}</p>
      </div>
    </div>
  );
}
