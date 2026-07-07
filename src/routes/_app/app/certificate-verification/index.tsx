import { createFileRoute } from '@tanstack/react-router';
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import certPlaceholder from '@/assets/certification/certplaceholder.png';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  CheckCircle2,
  XCircle,
  Loader2,
  FileText,
  Shield,
  Check,
  Copy,
  Clock,
  Database,
  Link as LinkIcon,
  QrCode,
} from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { QrScanner } from '@/components/QrScanner';
import { Skeleton } from '@/components/ui/skeleton';

export const Route = createFileRoute('/_app/app/certificate-verification/')({
  component: RouteComponent,
});

interface CertificateMetadata {
  id?: string | { $oid: string };
  sessionId: string;
  certificateId: string;
  recipientName: string;
  assessmentName: string;
  score: number;
  pdfHash: string;
  ipfsCID: string;
  ipfsURL: string;
  txHash: string;
  createdAt: string | { $date: string };
  updatedAt: string | { $date: string };
}

interface VerificationResult {
  isValid: boolean;
  status: string;
  message: string;
  metadata?: CertificateMetadata;
}

function RouteComponent() {
  const [selectedFile, setSelectedFile] = React.useState<File | null>(null);
  const [isVerifying, setIsVerifying] = React.useState(false);
  const [verificationResult, setVerificationResult] =
    useState<VerificationResult | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [qrScanned, setQrScanned] = useState(false);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    const file = e.dataTransfer.files?.[0];
    if (file && file.type === 'application/pdf') {
      setSelectedFile(file);
      setVerificationResult(null);
    }
  };

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const getFormattedDate = (dateVal: any) => {
    if (!dateVal) return '-';
    const dString =
      typeof dateVal === 'string'
        ? dateVal
        : dateVal.$date
          ? dateVal.$date
          : String(dateVal);
    try {
      return new Date(dString).toLocaleString('id-ID', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZoneName: 'short',
      });
    } catch {
      return dString;
    }
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file && file.type === 'application/pdf') {
      setSelectedFile(file);
      setVerificationResult(null);
    }
  };

  const verifyByFile = async () => {
    if (!selectedFile) return;

    setIsVerifying(true);
    setVerificationResult(null);

    try {
      const reader = new FileReader();
      reader.readAsArrayBuffer(selectedFile);

      reader.onload = async () => {
        try {
          const arrayBuffer = reader.result as ArrayBuffer;
          const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer);
          const hashArray = Array.from(new Uint8Array(hashBuffer));
          const pdfHash =
            '0x' +
            hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');

          console.log('[Verification] PDF Hash:', pdfHash);

          const response = await fetch('/api/blockchain/verify-by-hash', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            credentials: 'include',
            body: JSON.stringify({
              pdfHash: pdfHash,
            }),
          });

          if (!response.ok) {
            throw new Error('Failed to verify certificate with blockchain');
          }

          const data = await response.json();

          setVerificationResult({
            isValid: data.isValid,
            status: data.status,
            message: data.message,
            metadata: data.metadata,
          });
          setIsVerifying(false);
        } catch (error) {
          console.error('Error verifying certificate:', error);
          setVerificationResult({
            isValid: false,
            status: 'Error',
            message: `Gagal memverifikasi sertifikat: ${error instanceof Error ? error.message : 'Silakan coba lagi.'}`,
          });
          setIsVerifying(false);
        }
      };

      reader.onerror = () => {
        setVerificationResult({
          isValid: false,
          status: 'Error',
          message: 'Gagal membaca file PDF',
        });
        setIsVerifying(false);
      };
    } catch (error) {
      console.error('Error verifying certificate:', error);
      setVerificationResult({
        isValid: false,
        status: 'Error',
        message: `Gagal memverifikasi sertifikat: ${error instanceof Error ? error.message : 'Silakan coba lagi.'}`,
      });
      setIsVerifying(false);
    }
  };

  const verifyByCertificateId = async (certificateId: string) => {
    setQrScanned(true);
    setIsVerifying(true);
    setVerificationResult(null);

    try {
      const response = await fetch(
        '/api/blockchain/verify-by-certificate-id',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ certificateId }),
        },
      );

      if (!response.ok) {
        throw new Error('Failed to verify certificate');
      }

      const data = await response.json();

      setVerificationResult({
        isValid: data.isValid,
        status: data.status,
        message: data.message,
        metadata: data.metadata,
      });
    } catch (error) {
      setVerificationResult({
        isValid: false,
        status: 'Error',
        message: `Gagal memverifikasi sertifikat: ${error instanceof Error ? error.message : 'Silakan coba lagi.'}`,
      });
    } finally {
      setIsVerifying(false);
    }
  };

  const resetVerification = () => {
    setSelectedFile(null);
    setVerificationResult(null);
    setQrScanned(false);
  };

  return (
    <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="space-y-6">
        {/* Header Section */}
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Verifikasi Sertifikat
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Verifikasi keaslian sertifikat melalui blockchain
          </p>
        </div>

        {/* Verification Methods */}
        <Tabs defaultValue="upload" className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="upload" className="gap-2">
              <FileText className="h-4 w-4" />
              Upload PDF
            </TabsTrigger>
            <TabsTrigger value="qr" className="gap-2">
              <QrCode className="h-4 w-4" />
              Scan QR
            </TabsTrigger>
          </TabsList>

          <TabsContent value="upload">
            <Card>
              <CardHeader>
                <CardTitle>Upload Sertifikat</CardTitle>
                <CardDescription>
                  Upload file PDF sertifikat untuk memverifikasi keasliannya
                  melalui blockchain
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    className={`border-2 border-dashed rounded-lg p-8 text-center transition-all ${isDragging
                        ? 'border-indigo-500 bg-indigo-50/50 scale-[1.01]'
                        : 'border-muted-foreground/20 hover:border-primary/50'
                      }`}
                  >
                    <input
                      type="file"
                      accept="application/pdf"
                      onChange={handleFileUpload}
                      className="hidden"
                      id="file-upload"
                    />
                    <label
                      htmlFor="file-upload"
                      className="cursor-pointer flex flex-col items-center gap-3"
                    >
                      <div className="rounded-full bg-primary/10 p-4">
                        <FileText className="h-8 w-8 text-primary" />
                      </div>
                      <div>
                        <p
                          className="text-sm font-medium truncate max-w-[200px] sm:max-w-xs md:max-w-md mx-auto"
                          title={selectedFile ? selectedFile.name : ''}
                        >
                          {selectedFile
                            ? selectedFile.name
                            : isDragging
                              ? 'Lepaskan file di sini...'
                              : 'Klik atau seret file sertifikat PDF ke sini'}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          Format: PDF, Maksimal 5MB
                        </p>
                      </div>
                    </label>
                  </div>

                  {selectedFile && (
                    <>
                      <div className="flex gap-3">
                        <Button
                          onClick={verifyByFile}
                          disabled={isVerifying}
                          className="flex-1"
                        >
                          {isVerifying ? (
                            <>
                              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              Memverifikasi...
                            </>
                          ) : (
                            <>
                              <Shield className="mr-2 h-4 w-4" />
                              Verifikasi Sertifikat
                            </>
                          )}
                        </Button>
                        <Button
                          variant="outline"
                          onClick={resetVerification}
                          disabled={isVerifying}
                        >
                          Reset
                        </Button>
                      </div>
                    </>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="qr">
            <Card>
              <CardHeader>
                <CardTitle>Scan QR Code</CardTitle>
                <CardDescription>
                  Scan QR Code yang terdapat pada sertifikat untuk memverifikasi
                  keasliannya melalui blockchain
                </CardDescription>
              </CardHeader>
              <CardContent>
                {qrScanned ? (
                  <div className="flex flex-col items-center gap-4 py-4">
                    <p className="text-sm text-muted-foreground">
                      QR Code berhasil discan
                    </p>
                    <Button
                      variant="outline"
                      onClick={resetVerification}
                      className="gap-2"
                    >
                      <QrCode className="h-4 w-4" />
                      Scan Ulang
                    </Button>
                  </div>
                ) : isVerifying ? (
                  <div className="space-y-4">
                    <Skeleton className="h-[300px] w-full rounded-lg" />
                    <div className="flex items-center gap-2 justify-center">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span className="text-sm text-muted-foreground">
                        Memverifikasi...
                      </span>
                    </div>
                  </div>
                ) : (
                  <QrScanner
                    onScan={verifyByCertificateId}
                    onError={(err) => console.error('QR Scanner error:', err)}
                  />
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
        {/* Verification Result */}
        {verificationResult && (
          <div
            className={`relative border border-foreground/10 p-6 md:p-8 rounded-2xl overflow-hidden bg-card/40 backdrop-blur-md transition-all duration-300 `}
          >
            {/* Top Gradient Accent Line */}
            <div
              className={`absolute top-0 left-0 right-0 h-1 bg-gradient-to-r ${verificationResult.isValid
                  ? 'from-emerald-500 via-teal-500 to-emerald-300'
                  : 'from-rose-500 via-red-500 to-rose-300'
                }`}
            />

            <div className="flex items-start gap-3 mb-6">
              {verificationResult.isValid ? (
                <CheckCircle2 className="h-6 w-6 text-emerald-500 mt-0.5" />
              ) : (
                <XCircle className="h-6 w-6 text-rose-500 mt-0.5" />
              )}
              <div className="flex-1">
                <h3
                  className={`text-lg font-bold ${verificationResult.isValid
                      ? 'text-emerald-900 dark:text-emerald-400'
                      : 'text-rose-900 dark:text-rose-400'
                    }`}
                >
                  {verificationResult.isValid
                    ? 'Sertifikat Valid ✓'
                    : 'Sertifikat Tidak Valid ✗'}
                </h3>
              </div>
            </div>

            <div
              className={
                verificationResult.isValid && verificationResult.metadata
                  ? 'grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 items-start'
                  : 'space-y-6'
              }
            >
              {/* Left Column: Visual Certificate Card Preview */}
              {verificationResult.isValid && verificationResult.metadata && (
                <div className="lg:col-span-5 flex flex-col justify-start">
                  <div className="w-full overflow-hidden rounded-xl border border-foreground/10 bg-card/60 backdrop-blur-sm transition-all duration-300 hover:border-primary/30 hover:shadow-md">
                    {/* Header Image with Verified Badge */}
                    <div className="relative flex items-center justify-center overflow-hidden bg-muted/10 aspect-4/3">
                      <img
                        src={certPlaceholder}
                        alt={verificationResult.metadata.assessmentName}
                        className="w-full h-full object-cover"
                      />
                      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,var(--tw-gradient-stops))] from-primary/10 via-transparent to-transparent" />

                      {/* Verified Badge - Absolute Top Right */}
                      <div className="absolute top-3 right-3 z-20">
                        <Badge
                          variant="default"
                          className="bg-primary text-primary-foreground hover:bg-primary/95"
                        >
                          Verified
                        </Badge>
                      </div>
                    </div>

                    <div>
                      {/* Title Section */}
                      <div className="bg-muted/25 px-4 py-3 border-b border-foreground/5">
                        <h3 className="text-base font-bold text-foreground leading-tight">
                          {verificationResult.metadata.assessmentName}
                        </h3>
                        <p className="text-xs text-muted-foreground mt-1">
                          Penerima: {verificationResult.metadata.recipientName}
                        </p>
                      </div>

                      {/* Content Section */}
                      <div className="p-4 space-y-3">
                        {/* Score Display */}
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium text-muted-foreground">
                            Score
                          </span>
                          <span className="text-2xl font-bold text-foreground">
                            {verificationResult.metadata.score}
                            <span className="text-sm text-muted-foreground font-normal">
                              /1000
                            </span>
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Right Column: Status and Metadata */}
              <div
                className={
                  verificationResult.isValid && verificationResult.metadata
                    ? 'lg:col-span-7 space-y-6'
                    : 'space-y-6'
                }
              >
                <div className="rounded-xl border border-foreground/10 bg-card/20 p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-xs font-medium text-muted-foreground">
                        Status Blockchain
                      </p>
                      <p className="text-sm font-semibold mt-1 text-foreground">
                        {verificationResult.status}
                      </p>
                    </div>
                    <Shield
                      className={`h-8 w-8 ${verificationResult.isValid ? 'text-emerald-500' : 'text-rose-500'}`}
                    />
                  </div>
                </div>

                {verificationResult.isValid && verificationResult.metadata && (
                  <div className="rounded-xl border border-foreground/10 bg-card/20 p-5 space-y-4 text-left">
                    <div>
                      <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
                        <Database className="h-4.5 w-4.5 text-primary" />
                        Detail Metadata Blockchain
                      </h3>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Informasi verifikasi kriptografis permanen
                      </p>
                    </div>

                    <div className="space-y-3 divide-y divide-border">
                      {/* Recipient Name */}
                      {verificationResult.metadata.recipientName && (
                        <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                          <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                            Nama Lengkap
                          </span>
                          <span className="text-xs font-medium text-foreground">
                            {verificationResult.metadata.recipientName}
                          </span>
                        </div>
                      )}

                      {/* Assessment Name */}
                      {verificationResult.metadata.assessmentName && (
                        <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                          <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                            Assessment
                          </span>
                          <span className="text-xs font-medium text-foreground">
                            {verificationResult.metadata.assessmentName}
                          </span>
                        </div>
                      )}

                      {/* Score */}
                      {verificationResult.metadata.score > 0 && (
                        <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                          <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                            Score
                          </span>
                          <span className="text-xs font-semibold text-foreground">
                            {verificationResult.metadata.score}
                            <span className="text-muted-foreground font-normal">
                              {' '}
                              / 1000
                            </span>
                          </span>
                        </div>
                      )}

                      {/* Certificate ID */}
                      <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                        <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                          Certificate ID
                        </span>
                        <div className="flex items-center gap-1.5 flex-1 justify-end max-w-full overflow-hidden">
                          <span className="font-mono text-xs text-foreground break-all select-all">
                            {verificationResult.metadata.certificateId}
                          </span>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-muted-foreground hover:text-primary hover:bg-muted flex-shrink-0"
                            onClick={() =>
                              handleCopy(
                                verificationResult.metadata!.certificateId,
                                'certId',
                              )
                            }
                          >
                            {copiedField === 'certId' ? (
                              <Check className="h-3 w-3 text-emerald-500" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                      </div>

                      {/* Transaction Hash */}
                      <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                        <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                          Transaction Hash
                        </span>
                        <div className="flex items-center gap-1.5 flex-1 justify-end max-w-full overflow-hidden">
                          <span className="font-mono text-xs text-foreground break-all select-all">
                            {verificationResult.metadata.txHash || '-'}
                          </span>
                          {verificationResult.metadata.txHash && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6 text-muted-foreground hover:text-primary hover:bg-muted flex-shrink-0"
                              onClick={() =>
                                handleCopy(
                                  verificationResult.metadata!.txHash,
                                  'txHash',
                                )
                              }
                            >
                              {copiedField === 'txHash' ? (
                                <Check className="h-3 w-3 text-emerald-500" />
                              ) : (
                                <Copy className="h-3 w-3" />
                              )}
                            </Button>
                          )}
                        </div>
                      </div>

                      {/* IPFS CID & URL */}
                      <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                        <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                          IPFS CID & URL
                        </span>
                        <div className="flex flex-col items-end gap-1 flex-1 max-w-full overflow-hidden">
                          <div className="flex items-center gap-1.5 max-w-full">
                            <span className="font-mono text-xs text-foreground break-all select-all">
                              {verificationResult.metadata.ipfsCID}
                            </span>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6 text-muted-foreground hover:text-primary hover:bg-muted flex-shrink-0"
                              onClick={() =>
                                handleCopy(
                                  verificationResult.metadata!.ipfsCID,
                                  'ipfsCID',
                                )
                              }
                            >
                              {copiedField === 'ipfsCID' ? (
                                <Check className="h-3 w-3 text-emerald-500" />
                              ) : (
                                <Copy className="h-3 w-3" />
                              )}
                            </Button>
                          </div>
                        </div>
                      </div>

                      {/* PDF Hash */}
                      <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                        <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                          PDF Hash (SHA-256)
                        </span>
                        <div className="flex items-center gap-1.5 flex-1 justify-end max-w-full overflow-hidden">
                          <span className="font-mono text-[11px] text-foreground break-all select-all">
                            {verificationResult.metadata.pdfHash}
                          </span>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-muted-foreground hover:text-primary hover:bg-muted flex-shrink-0"
                            onClick={() =>
                              handleCopy(
                                verificationResult.metadata!.pdfHash,
                                'pdfHash',
                              )
                            }
                          >
                            {copiedField === 'pdfHash' ? (
                              <Check className="h-3 w-3 text-emerald-500" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                      </div>

                      {/* Created At */}
                      <div className="pt-2 flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                        <span className="text-xs font-semibold text-muted-foreground min-w-[140px]">
                          Waktu Dibuat
                        </span>
                        <div className="flex items-center gap-1.5 text-foreground justify-end">
                          <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                          <span className="text-xs font-medium">
                            {getFormattedDate(
                              verificationResult.metadata.createdAt,
                            )}
                          </span>
                        </div>
                      </div>
                    </div>
                    {/* Action Buttons */}
                    <div className="flex flex-col sm:flex-row gap-2 pt-3">
                      <a
                        href={`https://amoy.polygonscan.com/tx/${verificationResult.metadata.txHash}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex-1"
                      >
                        <Button className="w-full bg-primary hover:bg-primary/90 text-primary-foreground flex items-center justify-center gap-1.5 text-xs h-9">
                          <LinkIcon className="h-3.5 w-3.5" />
                          Buka Link Blockchain
                        </Button>
                      </a>
                      <a
                        href={verificationResult.metadata.ipfsURL}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex-1"
                      >
                        <Button
                          variant="outline"
                          className="w-full flex items-center justify-center gap-1.5 border-primary hover:bg-primary text-foreground text-xs h-9"
                        >
                          <FileText className="h-3.5 w-3.5" />
                          Lihat PDF Asli
                        </Button>
                      </a>
                    </div>
                  </div>
                )}

              </div>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
