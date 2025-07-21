import React from 'react';
import { Button } from '@/components/ui/button';
import { 
  isNativeApp, 
  openInFinder, 
  showNativeNotification, 
  requestPermission 
} from '@/lib/native-bridge';
import { Folder, Bell, Shield } from 'lucide-react';

/**
 * Example component showing how to use the native bridge
 * This demonstrates the JavaScript bridge functionality when running in the Mac app
 */
export function NativeBridgeExample() {
  const isNative = isNativeApp();

  const handleOpenInFinder = () => {
    const path = '/Users'; // Example path
    openInFinder(path);
  };

  const handleShowNotification = async () => {
    // Request permission first if needed
    const granted = await requestPermission('notifications');
    if (granted) {
      showNativeNotification(
        'Vibeman Notification',
        'This is a native macOS notification!'
      );
    } else {
      alert('Notification permission denied');
    }
  };

  if (!isNative) {
    return (
      <div className="p-4 border rounded-lg bg-muted">
        <p className="text-sm text-muted-foreground">
          Native bridge features are only available when running in the Vibeman Mac app.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold">Native Bridge Features</h3>
      
      <div className="space-y-2">
        <Button 
          onClick={handleOpenInFinder}
          variant="outline"
          className="w-full justify-start"
        >
          <Folder className="mr-2 h-4 w-4" />
          Open in Finder
        </Button>

        <Button 
          onClick={handleShowNotification}
          variant="outline"
          className="w-full justify-start"
        >
          <Bell className="mr-2 h-4 w-4" />
          Show Native Notification
        </Button>

        <Button 
          onClick={() => requestPermission('notifications')}
          variant="outline"
          className="w-full justify-start"
        >
          <Shield className="mr-2 h-4 w-4" />
          Request Notification Permission
        </Button>
      </div>

      <p className="text-sm text-muted-foreground">
        Running in native Mac app mode
      </p>
    </div>
  );
}

// Example of how to integrate with worktree or repository components
export function useNativeIntegration() {
  const openWorktreeInFinder = (worktreePath: string) => {
    if (isNativeApp()) {
      openInFinder(worktreePath);
    } else {
      // Fallback for web - maybe show the path to copy
      console.log('Path:', worktreePath);
    }
  };

  const notifyWorktreeReady = (worktreeName: string) => {
    if (isNativeApp()) {
      showNativeNotification(
        'Worktree Ready',
        `${worktreeName} is ready to use!`
      );
    }
  };

  return {
    openWorktreeInFinder,
    notifyWorktreeReady,
    isNativeApp: isNativeApp(),
  };
}