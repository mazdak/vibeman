/**
 * Utility functions for file downloads
 */

export interface DownloadOptions {
  filename: string;
  mimeType?: string;
  content: string | Blob;
}

/**
 * Download content as a file
 */
export function downloadFile(options: DownloadOptions): void {
  const { filename, mimeType = 'text/plain', content } = options;
  
  const blob = content instanceof Blob 
    ? content 
    : new Blob([content], { type: mimeType });
    
  const url = URL.createObjectURL(blob);
  
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.style.display = 'none';
  
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  
  // Revoke the URL after a short delay to ensure download starts
  setTimeout(() => {
    URL.revokeObjectURL(url);
  }, 100);
}

/**
 * Generate a timestamped filename
 */
export function generateTimestampedFilename(
  prefix: string, 
  extension: string, 
  date: Date = new Date()
): string {
  const timestamp = date.toISOString().split('T')[0]; // YYYY-MM-DD format
  return `${prefix}-${timestamp}.${extension}`;
}

/**
 * Download JSON data as a file
 */
export function downloadJSON(data: any, filename: string): void {
  downloadFile({
    filename: filename.endsWith('.json') ? filename : `${filename}.json`,
    mimeType: 'application/json',
    content: JSON.stringify(data, null, 2)
  });
}

/**
 * Download text content as a file
 */
export function downloadText(content: string, filename: string): void {
  downloadFile({
    filename: filename.endsWith('.txt') ? filename : `${filename}.txt`,
    mimeType: 'text/plain',
    content
  });
}