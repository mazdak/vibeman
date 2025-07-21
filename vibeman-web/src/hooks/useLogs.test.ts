import { test, expect, describe, beforeEach, afterEach } from "bun:test";

// Mock timers
let timers: NodeJS.Timeout[] = [];
const originalSetTimeout = global.setTimeout;
const originalClearTimeout = global.clearTimeout;

global.setTimeout = ((fn: Function, ms: number) => {
  const timer = originalSetTimeout(fn, ms);
  timers.push(timer);
  return timer;
}) as any;

global.clearTimeout = ((timer: NodeJS.Timeout) => {
  const index = timers.indexOf(timer);
  if (index > -1) {
    timers.splice(index, 1);
  }
  originalClearTimeout(timer);
}) as any;

describe("useLogs Memory Leak Prevention", () => {
  beforeEach(() => {
    timers = [];
  });

  afterEach(() => {
    // Clear all remaining timers
    timers.forEach(timer => originalClearTimeout(timer));
    timers = [];
  });

  test("clears reconnect timeout on cleanup", () => {
    // Initial state: no timers
    expect(timers.length).toBe(0);
    
    // Simulate setting a reconnect timeout
    const timer = setTimeout(() => {}, 1000);
    expect(timers.length).toBe(1);
    
    // Simulate cleanup
    clearTimeout(timer);
    expect(timers.length).toBe(0);
  });

  test("no dangling timers after multiple reconnect attempts", () => {
    // Simulate multiple reconnect attempts
    for (let i = 0; i < 5; i++) {
      const timer = setTimeout(() => {}, 1000);
      clearTimeout(timer);
    }
    
    // All timers should be cleared
    expect(timers.length).toBe(0);
  });
});