import { Scanner } from '@yudiel/react-qr-scanner';

interface QrScannerProps {
  onScan: (data: string) => void;
  onError?: (error: string) => void;
}

export function QrScanner({ onScan, onError }: QrScannerProps) {
  return (
    <div className="mx-auto max-w-xs">
      <Scanner
        onScan={(detectedCodes) => {
          if (detectedCodes && detectedCodes.length > 0) {
            onScan(detectedCodes[0].rawValue);
          }
        }}
        onError={(error) => {
          onError?.(error?.message || 'Camera error');
        }}
        constraints={{ facingMode: 'environment' }}
        components={{ finder: true }}
        styles={{
          container: { borderRadius: '0.5rem', overflow: 'hidden' },
        }}
      />
    </div>
  );
}
