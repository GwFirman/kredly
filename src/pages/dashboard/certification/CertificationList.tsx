import * as React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { Search, Award, ChevronDown, ChevronUp, Loader2 } from 'lucide-react';
import { Link, useNavigate } from '@tanstack/react-router';
import certPlaceholder from '@/assets/certification/certplaceholder.png';
import { sessionService } from '@/services/sessionService';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

interface CredentialAttempt {
  id: string;
  score: number;
  dateEarned: string;
  blockchainTxHash: string;
  sessionId: string;
  level: string;
  createdAt: string;
}

interface GroupedCredential {
  skillName: string;
  bestScore: number;
  bestLevel: string;
  attempts: CredentialAttempt[];
}

interface CertificationListProps {
  groupedCredentials: GroupedCredential[];
  viewMode: 'grid' | 'list';
  searchQuery: string;
  filterStatus: 'all' | 'verified';
  setSearchQuery: (value: string) => void;
  setFilterStatus: (value: 'all' | 'verified') => void;
}

export default function CertificationList({
  groupedCredentials,
  viewMode,
  searchQuery,
  filterStatus,
  setSearchQuery,
  setFilterStatus,
}: CertificationListProps) {
  const navigate = useNavigate();
  const [downloadingId, setDownloadingId] = React.useState<string | null>(null);
  const [expandedSkills, setExpandedSkills] = React.useState<{
    [key: string]: boolean;
  }>({});
  const [retakeSkill, setRetakeSkill] =
    React.useState<GroupedCredential | null>(null);
  const [isRetaking, setIsRetaking] = React.useState(false);

  const toggleExpand = (e: React.MouseEvent, skillName: string) => {
    e.preventDefault();
    e.stopPropagation();
    setExpandedSkills((prev) => ({
      ...prev,
      [skillName]: !prev[skillName],
    }));
  };

  const handleDownload = async (e: React.MouseEvent, sessionId?: string) => {
    e.preventDefault();
    e.stopPropagation();
    if (!sessionId) {
      alert('Session ID tidak ditemukan.');
      return;
    }

    setDownloadingId(sessionId);
    try {
      const response = await fetch(`/api/certificates/metadata/${sessionId}`);
      if (response.ok) {
        const data = await response.json();
        if (data.exists && data.metadata && data.metadata.ipfsURL) {
          window.open(data.metadata.ipfsURL, '_blank');
          return;
        }
      }
      alert('Sertifikat belum tersedia atau sedang diproses di blockchain.');
    } catch (err) {
      console.error('Failed to fetch certificate metadata:', err);
      alert('Gagal mengambil detail sertifikat.');
    } finally {
      setDownloadingId(null);
    }
  };

  const handleDetailClick = (e: React.MouseEvent, sessionId?: string) => {
    e.preventDefault();
    e.stopPropagation();
    if (!sessionId) return;
    navigate({
      to: '/app/certification/$id',
      params: { id: sessionId },
    });
  };

  const handleConfirmRetake = async () => {
    if (!retakeSkill) return;
    setIsRetaking(true);

    try {
      // 1. Fetch user profile config to get specific assessment configuration
      const profileRes = await fetch('/api/profile', {
        credentials: 'include',
      });
      if (!profileRes.ok) throw new Error('Gagal mengambil konfigurasi profil');
      const profileData = await profileRes.json();
      const cvAssessments = profileData?.profile?.cvAssessments || [];

      // Find the matching assessment configuration
      const assessment = cvAssessments.find(
        (a: any) => a.title === retakeSkill.skillName,
      );
      if (!assessment) {
        throw new Error('Konfigurasi asesmen untuk skill ini tidak ditemukan.');
      }

      // 2. Call session creation service directly
      const sessionResponse = await sessionService.createSession({
        role: assessment.title,
        level: profileData?.profile?.cvLevel || 'Junior',
        skills:
          assessment.type === 'general'
            ? assessment.topics || []
            : [assessment.title],
        cv_summary:
          profileData?.profile?.cvSummary || `${assessment.title} assessment`,
        assessment_id: assessment.id,
      });

      // 3. Immediately redirect to the active exam session
      navigate({
        to: '/quiz/$sessionId',
        params: { sessionId: sessionResponse.session_id },
      });
    } catch (err: any) {
      console.error('Failed to start retake:', err);
      alert(err.message || 'Gagal memulai sesi asesmen baru.');
    } finally {
      setIsRetaking(false);
      setRetakeSkill(null);
    }
  };

  if (groupedCredentials.length > 0) {
    return (
      <>
        <div
          className={
            viewMode === 'grid'
              ? 'grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3'
              : 'space-y-4'
          }
        >
          {groupedCredentials.map((item) => {
            // Find the best attempt's details (for download/view details on the card header)
            const bestAttempt =
              item.attempts.find((a) => a.score === item.bestScore) ||
              item.attempts[0];
            const bestSessionId = bestAttempt.sessionId;
            const latestAttempt = item.attempts[0];
            const isExpanded = !!expandedSkills[item.skillName];

            return (
              <div
                key={item.skillName}
                className={`overflow-hidden rounded-xl border border-gray-200 bg-white transition-all hover:shadow-sm ${viewMode === 'list' ? 'flex flex-row' : ''
                  }`}
              >
                {/* Header Image with Verified Badge */}
                <div
                  className={`relative flex items-center justify-center overflow-hidden bg-gray-100 ${viewMode === 'list'
                    ? 'w-48 shrink-0 aspect-4/3'
                    : 'w-full aspect-4/3'
                    }`}
                >
                  <img
                    src={certPlaceholder}
                    alt={item.skillName}
                    className="w-full h-full object-cover"
                  />
                  <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,var(--tw-gradient-stops))] from-white/10 via-transparent to-transparent" />

                  {/* Verified Badge - Absolute Top Right */}
                  <div className="absolute top-3 right-3 z-20">
                    <Badge variant="default">Verified</Badge>
                  </div>
                </div>

                <div
                  className={`flex-1 flex flex-col justify-between ${viewMode === 'list' ? '' : ''}`}
                >
                  <div>
                    {/* Gray Title Section */}
                    <div className="bg-gray-50 px-4 py-3 border-b border-gray-100 flex justify-between items-center">
                      <div>
                        <h3 className="text-base font-bold text-gray-900 leading-tight">
                          {item.skillName}
                        </h3>
                        <p className="text-xs text-gray-500 mt-1">
                          Terakhir: {latestAttempt.dateEarned}
                        </p>
                      </div>
                      <Badge variant="secondary" className="text-xs">
                        {item.bestLevel}
                      </Badge>
                    </div>

                    {/* Content Section */}
                    <div className="p-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-gray-600">
                          Skor Terbaik
                        </span>
                        <span className="text-2xl font-bold text-gray-900">
                          {item.bestScore}
                          <span className="text-sm text-gray-500 font-normal">
                            {item.bestScore > 100 ? '/1000' : '/100'}
                          </span>
                        </span>
                      </div>

                      {/* Accordion / History Trigger */}
                      <div className="pt-1">
                        <button
                          onClick={(e) => toggleExpand(e, item.skillName)}
                          className="flex items-center gap-1 text-xs font-semibold text-primary hover:text-primary/80 transition-colors"
                        >
                          {isExpanded ? (
                            <>
                              <ChevronUp className="h-4 w-4" />
                              Sembunyikan Riwayat Percobaan (
                              {item.attempts.length})
                            </>
                          ) : (
                            <>
                              <ChevronDown className="h-4 w-4" />
                              Lihat Riwayat Percobaan
                            </>
                          )}
                        </button>
                      </div>

                      {/* Accordion Content */}
                      {isExpanded && (
                        <div className="mt-2 pt-2 border-t border-gray-100 space-y-2 max-h-48 overflow-y-auto pr-1">
                          {item.attempts.map((attempt, index) => (
                            <div
                              key={attempt.id}
                              className="flex items-center justify-between text-xs p-2 bg-gray-50 rounded-lg border border-gray-100 hover:bg-gray-100/50 transition-colors"
                            >
                              <div className="flex flex-col">
                                <span className="font-semibold text-gray-700">
                                  Percobaan #{item.attempts.length - index}
                                </span>
                                <span className="text-[10px] text-gray-500">
                                  {attempt.dateEarned}
                                </span>
                              </div>
                              <div className="flex items-center gap-3">
                                <span className="font-bold text-gray-800">
                                  {attempt.score}
                                </span>
                                <div className="flex gap-1">
                                  <Button
                                    size="xs"
                                    variant="outline"
                                    onClick={(e) =>
                                      handleDownload(e, attempt.sessionId)
                                    }
                                    disabled={
                                      downloadingId === attempt.sessionId
                                    }
                                    className="h-7 px-2 text-[10px]"
                                  >
                                    {downloadingId === attempt.sessionId
                                      ? '...'
                                      : 'Unduh'}
                                  </Button>
                                  <Button
                                    size="xs"
                                    variant="outline"
                                    onClick={(e) =>
                                      handleDetailClick(e, attempt.sessionId)
                                    }
                                    className="h-7 px-2 text-[10px]"
                                  >
                                    Detail
                                  </Button>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Actions Section */}
                  <div className="px-4 pb-4 pt-1 flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1 text-xs"
                      onClick={(e) => handleDownload(e, bestSessionId)}
                      disabled={downloadingId === bestSessionId}
                    >
                      {downloadingId === bestSessionId
                        ? 'Memuat...'
                        : 'Unduh Terbaik'}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1 text-xs"
                      onClick={(e) => handleDetailClick(e, bestSessionId)}
                    >
                      Detail Terbaik
                    </Button>
                    <Button
                      size="sm"
                      variant="default"
                      className="flex-1 text-xs bg-primary hover:bg-primary/70 text-white font-medium"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setRetakeSkill(item);
                      }}
                      disabled={isRetaking}
                    >
                      Uji Ulang
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Retake Confirmation Dialog */}
        <AlertDialog
          open={!!retakeSkill}
          onOpenChange={(open) => !open && setRetakeSkill(null)}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Mulai Uji Ulang Asesmen?</AlertDialogTitle>
              <AlertDialogDescription className="space-y-2 text-sm text-gray-600">
                <p>
                  Anda akan memulai sesi asesmen baru untuk{' '}
                  <strong className="text-gray-900">
                    {retakeSkill?.skillName}
                  </strong>
                  .
                </p>
                <ul className="list-disc pl-5 space-y-1 mt-2 text-xs">
                  <li>
                    Tindakan ini akan mengonsumsi{' '}
                    <strong>1 Kredit Token</strong> Anda.
                  </li>
                  <li>
                    Skor terbaik Anda sebelumnya akan tetap dipertahankan dan
                    ditampilkan sebagai sertifikat utama.
                  </li>
                  <li>
                    Setelah selesai, sertifikat baru akan diterbitkan dan
                    tercatat dalam riwayat percobaan.
                  </li>
                </ul>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={isRetaking}>Batal</AlertDialogCancel>
              <AlertDialogAction
                onClick={(e) => {
                  e.preventDefault();
                  handleConfirmRetake();
                }}
                disabled={isRetaking}
                className="bg-amber-600 hover:bg-amber-700 text-white flex items-center gap-1.5"
              >
                {isRetaking && <Loader2 className="h-4 w-4 animate-spin" />}
                {isRetaking ? 'Memulai...' : 'Ya, Mulai Sekarang'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </>
    );
  }

  if (searchQuery || filterStatus !== 'all') {
    return (
      <Empty className="bg-white">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Search />
          </EmptyMedia>
          <EmptyTitle>Tidak Ada Hasil</EmptyTitle>
          <EmptyDescription>
            Tidak ada kredensial yang sesuai dengan filter pencarian Anda. Coba
            ubah kata kunci atau filter.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button
            variant="default"
            onClick={() => {
              setSearchQuery('');
              setFilterStatus('all');
            }}
            size="sm"
          >
            Reset Filter
          </Button>
        </EmptyContent>
      </Empty>
    );
  }

  return (
    <Empty className="bg-white">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Award />
        </EmptyMedia>
        <EmptyTitle>Belum Ada Kredensial</EmptyTitle>
        <EmptyDescription>
          Selesaikan assessment untuk mendapatkan kredensial pertama Anda! Semua
          kredensial akan tersimpan di blockchain dan dapat diverifikasi.
        </EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Link to="/app/assessment">
          <Button size="sm">Mulai Assessment</Button>
        </Link>
      </EmptyContent>
    </Empty>
  );
}
