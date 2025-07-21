import { Play, Pause, RefreshCw, AlertCircle } from 'lucide-react';

export function getStatusIcon(status: string) {
  switch (status) {
    case 'running':
      return Play;
    case 'stopped':
      return Pause;
    case 'creating':
    case 'starting':
    case 'stopping':
      return RefreshCw;
    default:
      return AlertCircle;
  }
}

export function getStatusColor(status: string) {
  switch (status) {
    case 'running':
      return 'text-green-500';
    case 'stopped':
      return 'text-gray-500';
    case 'creating':
    case 'starting':
    case 'stopping':
      return 'text-blue-500';
    case 'error':
      return 'text-red-500';
    default:
      return 'text-gray-500';
  }
}

export function getStatusLabel(status: string): string {
  switch (status) {
    case 'running':
      return 'Running';
    case 'stopped':
      return 'Stopped';
    case 'creating':
      return 'Creating...';
    case 'starting':
      return 'Starting...';
    case 'stopping':
      return 'Stopping...';
    case 'error':
      return 'Error';
    default:
      return status.charAt(0).toUpperCase() + status.slice(1);
  }
}