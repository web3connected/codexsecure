'use client';

import React, { ReactNode } from 'react';
import { GlobalWrapper } from '@/components/codex/shared/GlobalWrapper';
import CodexSecureHeader from '@/components/layout/CodexSecureHeader';
import CodexSecureFooter from '@/components/layout/CodexSecureFooter';

interface CodexSecureDataLayerProps {
  children: ReactNode;
}

const CodexSecureDataLayer: React.FC<CodexSecureDataLayerProps> = ({ children }) => {
  return (
    <GlobalWrapper
      header={<CodexSecureHeader />}
      footer={<CodexSecureFooter />}
    >
      {children}
    </GlobalWrapper>
  );
};

export default CodexSecureDataLayer;
