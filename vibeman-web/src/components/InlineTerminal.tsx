import React from 'react';
import { Terminal } from './Terminal';
import type { TerminalProps } from '../types/terminal';
import { Card } from './ui/card';

interface InlineTerminalProps extends TerminalProps {
  title?: string;
  height?: string;
}

export const InlineTerminal: React.FC<InlineTerminalProps> = ({
  environmentId,
  title,
  height = '400px',
  className = '',
  onClose
}) => {
  return (
    <Card className={`overflow-hidden ${className}`}>
      <div className="p-0" style={{ height }}>
        <Terminal
          environmentId={environmentId}
          onClose={onClose}
          className="h-full border-0 rounded-lg"
        />
      </div>
    </Card>
  );
};

export default InlineTerminal;