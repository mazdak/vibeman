/**
 * Utility functions for managing reconnection logic
 */

export interface ReconnectionOptions {
  maxAttempts?: number;
  delay?: number;
  backoffMultiplier?: number;
  maxDelay?: number;
}

export class ReconnectionManager {
  private attempts = 0;
  private timeoutId: NodeJS.Timeout | null = null;
  private options: Required<ReconnectionOptions>;

  constructor(options: ReconnectionOptions = {}) {
    this.options = {
      maxAttempts: options.maxAttempts ?? 5,
      delay: options.delay ?? 2000,
      backoffMultiplier: options.backoffMultiplier ?? 1.5,
      maxDelay: options.maxDelay ?? 30000
    };
  }

  /**
   * Schedule a reconnection attempt
   */
  scheduleReconnect(callback: () => void): boolean {
    if (this.attempts >= this.options.maxAttempts) {
      return false;
    }

    this.attempts++;
    
    // Calculate delay with exponential backoff
    const delay = Math.min(
      this.options.delay * Math.pow(this.options.backoffMultiplier, this.attempts - 1),
      this.options.maxDelay
    );

    this.timeoutId = setTimeout(() => {
      this.timeoutId = null;
      callback();
    }, delay);

    return true;
  }

  /**
   * Reset the reconnection attempts counter
   */
  reset(): void {
    this.attempts = 0;
    this.clearTimeout();
  }

  /**
   * Cancel any pending reconnection
   */
  cancel(): void {
    this.clearTimeout();
  }

  /**
   * Get current attempt count
   */
  getAttempts(): number {
    return this.attempts;
  }

  /**
   * Check if max attempts reached
   */
  hasExceededMaxAttempts(): boolean {
    return this.attempts >= this.options.maxAttempts;
  }

  private clearTimeout(): void {
    if (this.timeoutId) {
      clearTimeout(this.timeoutId);
      this.timeoutId = null;
    }
  }
}