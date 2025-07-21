/**
 * Native bridge interface for communication between the web UI and macOS app
 * This is injected by the Swift WebViewController when running inside the Mac app
 */

interface VibemanBridge {
  postMessage: (message: any) => void;
  openInFinder: (path: string) => void;
  showNotification: (title: string, body: string) => void;
  requestPermission: (permission: string) => void;
  onPermissionResult?: (permission: string, granted: boolean) => void;
}

declare global {
  interface Window {
    vibemanBridge?: VibemanBridge;
    webkit?: {
      messageHandlers?: {
        vibemanBridge?: {
          postMessage: (message: any) => void;
        };
      };
    };
  }
}

/**
 * Check if we're running inside the native Mac app
 */
export function isNativeApp(): boolean {
  return typeof window.vibemanBridge !== 'undefined';
}

/**
 * Open a file or folder in Finder (macOS only)
 * @param path The file system path to open
 */
export function openInFinder(path: string): void {
  if (window.vibemanBridge?.openInFinder) {
    window.vibemanBridge.openInFinder(path);
  } else {
    console.warn('openInFinder: Not running in native app');
  }
}

/**
 * Show a native notification (requires permission)
 * @param title Notification title
 * @param body Notification body text
 */
export function showNativeNotification(title: string, body: string): void {
  if (window.vibemanBridge?.showNotification) {
    window.vibemanBridge.showNotification(title, body);
  } else if ('Notification' in window) {
    // Fallback to web notifications
    if (Notification.permission === 'granted') {
      new Notification(title, { body });
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(permission => {
        if (permission === 'granted') {
          new Notification(title, { body });
        }
      });
    }
  }
}

/**
 * Request a system permission
 * @param permission The permission to request (e.g., 'notifications')
 * @returns Promise that resolves with whether permission was granted
 */
export function requestPermission(permission: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (window.vibemanBridge?.requestPermission) {
      // Set up callback for permission result
      window.vibemanBridge.onPermissionResult = (perm: string, granted: boolean) => {
        if (perm === permission) {
          resolve(granted);
          // Clean up callback
          if (window.vibemanBridge) {
            window.vibemanBridge.onPermissionResult = undefined;
          }
        }
      };
      window.vibemanBridge.requestPermission(permission);
    } else {
      // Not in native app, resolve based on web permissions
      if (permission === 'notifications' && 'Notification' in window) {
        if (Notification.permission === 'granted') {
          resolve(true);
        } else if (Notification.permission === 'denied') {
          resolve(false);
        } else {
          Notification.requestPermission().then(perm => {
            resolve(perm === 'granted');
          });
        }
      } else {
        resolve(false);
      }
    }
  });
}

/**
 * Send a custom message to the native app
 * @param type Message type
 * @param data Additional data
 */
export function sendToNative(type: string, data: any = {}): void {
  if (window.vibemanBridge?.postMessage) {
    window.vibemanBridge.postMessage({ type, ...data });
  } else {
    console.warn(`sendToNative: Not running in native app (type: ${type})`);
  }
}

// Export a default object for convenience
export default {
  isNativeApp,
  openInFinder,
  showNativeNotification,
  requestPermission,
  sendToNative,
};