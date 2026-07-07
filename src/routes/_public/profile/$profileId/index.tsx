import { createFileRoute, useParams } from '@tanstack/react-router';
import { useEffect, useState, useMemo } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ExternalLink, Award, FileText, Briefcase, Target } from 'lucide-react';
import { ProfileSkeleton } from '@/components/skeletons/ProfileSkeleton';
import PublicCertificateCard from '@/components/PublicCertificateCard';
import { Navbar } from '@/components/Navbar';
import Footer from '@/components/Footer';

export const Route = createFileRoute('/_public/profile/$profileId/')({
  component: RouteComponent,
});

interface PublicProfile {
  id: string;
  name: string;
  username: string;
  image?: string;
  headline?: string;
  bio?: string;
  cvRole?: string;
  cvLevel?: string;
  cvSkills?: string[];
  certificates?: Array<{
    id: string;
    sessionId?: string;
    title: string;
    score?: number;
    issuedAt: string;
    verificationUrl?: string;
  }>;
  assessments?: any[];
  socialLinks?: {
    linkedin?: string;
    github?: string;
    portfolio?: string;
    twitter?: string;
  };
  displaySettings?: {
    showCertificates: boolean;
    showAssessments: boolean;
    showSkills: boolean;
    showCVData: boolean;
  };
}

function RouteComponent() {
  const { profileId } = useParams({ from: '/_public/profile/$profileId/' });
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [certificatesPage, setCertificatesPage] = useState(1);
  const certificatesPerPage = 6;

  useEffect(() => {
    async function fetchProfile() {
      try {
        const response = await fetch(`/api/profile/public/${profileId}`);

        if (response.status === 404) {
          setNotFound(true);
          return;
        }

        if (!response.ok) {
          throw new Error('Failed to fetch profile');
        }

        const data = await response.json();
        setProfile(data.profile);
      } catch (error) {
        console.error('Error fetching profile:', error);
        setNotFound(true);
      } finally {
        setLoading(false);
      }
    }

    fetchProfile();
  }, [profileId]);

  // Group and keep only the best score certificate per skill/title
  const uniqueCertificates = useMemo(() => {
    if (!profile || !profile.certificates) return [];
    const bestCerts: { [title: string]: typeof profile.certificates[0] } = {};
    
    for (const cert of profile.certificates) {
      const existing = bestCerts[cert.title];
      if (!existing || (cert.score || 0) > (existing.score || 0)) {
        bestCerts[cert.title] = cert;
      } else if (cert.score === existing.score) {
        if (new Date(cert.issuedAt).getTime() > new Date(existing.issuedAt).getTime()) {
          bestCerts[cert.title] = cert;
        }
      }
    }
    
    return Object.values(bestCerts).sort(
      (a, b) => new Date(b.issuedAt).getTime() - new Date(a.issuedAt).getTime()
    );
  }, [profile?.certificates]);

  // Group and keep only the best score assessment per title
  const uniqueAssessments = useMemo(() => {
    if (!profile || !profile.assessments) return [];
    const bestAssessments: { [title: string]: typeof profile.assessments[0] } = {};
    
    for (const assess of profile.assessments) {
      const existing = bestAssessments[assess.title];
      if (!existing || (assess.score || 0) > (existing.score || 0)) {
        bestAssessments[assess.title] = assess;
      } else if (assess.score === existing.score) {
        if (new Date(assess.completedAt).getTime() > new Date(existing.completedAt).getTime()) {
          bestAssessments[assess.title] = assess;
        }
      }
    }
    
    return Object.values(bestAssessments).sort(
      (a, b) => new Date(b.completedAt).getTime() - new Date(a.completedAt).getTime()
    );
  }, [profile?.assessments]);

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  if (loading) {
    return (
      <>
        <Navbar />
        <ProfileSkeleton />
        <Footer />
      </>
    );
  }

  if (notFound || !profile) {
    return (
      <>
        <Navbar />
        <div className="min-h-screen flex items-center justify-center px-4">
          <div className="text-center max-w-md">
            <h1 className="text-4xl font-bold mb-4">404</h1>
            <p className="text-xl text-muted-foreground mb-6">
              Profil tidak ditemukan
            </p>
            <p className="text-sm text-muted-foreground mb-8">
              User @{profileId} tidak ada atau profil belum diatur sebagai
              publik
            </p>
            <Button asChild>
              <a href="/">Kembali ke Beranda</a>
            </Button>
          </div>
        </div>
        <Footer />
      </>
    );
  }

  return (
    <>
      <Navbar />
      <div className="p-6 md:p-8">
        <div className="mx-auto max-w-5xl">
          <Card className="overflow-hidden shadow-none">
            <CardContent className="p-0">
              {/* Header Section */}
              <div className="p-6 bg-gradient-to-br from-background to-muted/20">
                <div className="flex flex-col sm:flex-row items-start gap-6">
                  <Avatar className="h-24 w-24 border-2 border-primary/20">
                    <AvatarImage src={profile.image} alt={profile.name} />
                    <AvatarFallback className="text-2xl bg-primary/10 text-primary font-semibold">
                      {getInitials(profile.name)}
                    </AvatarFallback>
                  </Avatar>

                  <div className="flex-1 min-w-0">
                    <h1 className="text-2xl md:text-3xl font-bold mb-1">
                      {profile.name}
                    </h1>
                    <p className="text-muted-foreground mb-3">
                      @{profile.username}
                    </p>

                    {profile.headline && (
                      <p className="text-base font-medium text-foreground mb-3">
                        {profile.headline}
                      </p>
                    )}

                    {profile.bio && (
                      <p className="text-sm text-muted-foreground leading-relaxed">
                        {profile.bio}
                      </p>
                    )}
                  </div>
                </div>

                {/* Social Links */}
                {profile.socialLinks &&
                  (profile.socialLinks.linkedin ||
                    profile.socialLinks.github ||
                    profile.socialLinks.portfolio ||
                    profile.socialLinks.twitter) && (
                    <div className="mt-4 flex flex-wrap gap-2">
                      {profile.socialLinks.linkedin && (
                        <Button variant="outline" size="sm" asChild>
                          <a
                            href={profile.socialLinks.linkedin}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            LinkedIn
                            <ExternalLink className="ml-1.5 h-3 w-3" />
                          </a>
                        </Button>
                      )}
                      {profile.socialLinks.github && (
                        <Button variant="outline" size="sm" asChild>
                          <a
                            href={profile.socialLinks.github}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            GitHub
                            <ExternalLink className="ml-1.5 h-3 w-3" />
                          </a>
                        </Button>
                      )}
                      {profile.socialLinks.portfolio && (
                        <Button variant="outline" size="sm" asChild>
                          <a
                            href={profile.socialLinks.portfolio}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            Portfolio
                            <ExternalLink className="ml-1.5 h-3 w-3" />
                          </a>
                        </Button>
                      )}
                      {profile.socialLinks.twitter && (
                        <Button variant="outline" size="sm" asChild>
                          <a
                            href={profile.socialLinks.twitter}
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            Twitter
                            <ExternalLink className="ml-1.5 h-3 w-3" />
                          </a>
                        </Button>
                      )}
                    </div>
                  )}
              </div>

              <Separator />

              <Separator />

              {/* Assessments */}
              {uniqueAssessments.length > 0 &&
                profile.displaySettings?.showAssessments && (
                  <>
                    <div className="p-5">
                      <div className="flex items-center gap-2 mb-4">
                        <FileText className="h-4 w-4 text-muted-foreground" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                          Assessment Selesai
                        </h3>
                      </div>
                      <div className="space-y-2">
                        {uniqueAssessments.map((assessment: any) => (
                          <div
                            key={assessment.id}
                            className="flex items-center justify-between p-3 border rounded-lg hover:border-primary/50 transition-colors"
                          >
                            <div className="flex items-center gap-3 min-w-0 flex-1">
                              <div className="h-9 w-9 bg-primary/10 flex items-center justify-center rounded shrink-0">
                                <FileText className="h-4 w-4 text-primary" />
                              </div>
                              <div className="min-w-0 flex-1">
                                <p className="text-sm font-medium truncate">
                                  {assessment.title}
                                </p>
                                <p className="text-xs text-muted-foreground">
                                  {new Date(
                                    assessment.completedAt,
                                  ).toLocaleDateString('id-ID', {
                                    day: 'numeric',
                                    month: 'short',
                                    year: 'numeric',
                                  })}
                                </p>
                              </div>
                            </div>
                            {assessment.score !== undefined && (
                              <Badge
                                variant="default"
                                className="ml-2 shrink-0"
                              >
                                {assessment.score}%
                              </Badge>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                    <Separator />
                  </>
                )}

              {/* Professional Info */}
              {(profile.cvRole || profile.cvLevel) &&
                profile.displaySettings?.showCVData && (
                  <>
                    <div className="p-5">
                      <div className="flex items-center gap-2 mb-4">
                        <Briefcase className="h-4 w-4 text-muted-foreground" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                          Profile
                        </h3>
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        {profile.cvRole && (
                          <div className="p-3 bg-muted/30 rounded-lg">
                            <p className="text-xs text-muted-foreground mb-1 font-medium">
                              Role
                            </p>
                            <p className="text-sm font-semibold">
                              {profile.cvRole}
                            </p>
                          </div>
                        )}
                        {profile.cvLevel && (
                          <div className="p-3 bg-muted/30 rounded-lg">
                            <p className="text-xs text-muted-foreground mb-1 font-medium">
                              Level
                            </p>
                            <p className="text-sm font-semibold">
                              {profile.cvLevel}
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                    <Separator />
                  </>
                )}

              {/* Skills */}
              {profile.cvSkills &&
                profile.cvSkills.length > 0 &&
                profile.displaySettings?.showSkills && (
                  <>
                    <div className="p-5">
                      <div className="flex items-center gap-2 mb-4">
                        <Target className="h-4 w-4 text-muted-foreground" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                          Skills
                        </h3>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {profile.cvSkills.map((skill, idx) => (
                          <Badge
                            key={idx}
                            variant="default"
                            className="text-sm"
                          >
                            {skill}
                          </Badge>
                        ))}
                      </div>
                    </div>
                    <Separator />
                  </>
                )}

              {/* Certificates */}
              {uniqueCertificates.length > 0 &&
                profile.displaySettings?.showCertificates && (
                  <div className="p-5">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-2">
                        <Award className="h-4 w-4 text-muted-foreground" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                          Sertifikat Terverifikasi
                        </h3>
                      </div>
                      {uniqueCertificates.length > certificatesPerPage && (
                        <span className="text-xs text-muted-foreground">
                          Menampilkan{' '}
                          {Math.min(
                            certificatesPage * certificatesPerPage,
                            uniqueCertificates.length,
                          )}{' '}
                          dari {uniqueCertificates.length}
                        </span>
                      )}
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {uniqueCertificates
                        .slice(0, certificatesPage * certificatesPerPage)
                        .map((cert) => (
                          <PublicCertificateCard
                            key={cert.id}
                            certificate={cert}
                            viewMode="grid"
                          />
                        ))}
                    </div>
                    {uniqueCertificates.length >
                      certificatesPage * certificatesPerPage && (
                      <div className="mt-4 text-center">
                        <Button
                          variant="outline"
                          onClick={() =>
                            setCertificatesPage((prev) => prev + 1)
                          }
                        >
                          Lihat Lebih Banyak (
                          {uniqueCertificates.length -
                            certificatesPage * certificatesPerPage}{' '}
                          lagi)
                        </Button>
                      </div>
                    )}
                  </div>
                )}
            </CardContent>
          </Card>
        </div>
      </div>
      <Footer />
    </>
  );
}
