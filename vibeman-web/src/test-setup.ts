import '@testing-library/jest-dom';
import '@happy-dom/global-registrator';

// Setup DOM environment for React Testing Library
import { GlobalRegistrator } from '@happy-dom/global-registrator';

// Register the global DOM environment
GlobalRegistrator.register();

// Setup default window properties that tests might expect
if (typeof window !== 'undefined') {
  // Mock window.location if not already set
  if (!window.location) {
    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:3000',
        origin: 'http://localhost:3000'
      },
      writable: true
    });
  }

  // Mock window.matchMedia for responsive tests
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => {},
    }),
  });

  // Mock window.innerWidth for mobile tests
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: 1024
  });

  // Mock window.innerHeight for mobile tests  
  Object.defineProperty(window, 'innerHeight', {
    writable: true,
    configurable: true,
    value: 768
  });
}