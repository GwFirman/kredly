import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { Card, CardContent } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AssessmentCard } from '@/pages/dashboard/assessment/AssessmentCard';
import { GeneralAssessmentCard } from '@/pages/dashboard/assessment/GeneralAssessmentCard';
import { RelatedSkilAsessmentsSection } from '@/pages/dashboard/assessment/RelatedSkilAsessmentsSection';
import { CustomizationAndReuploadSection } from '@/pages/dashboard/assessment/CustomizationAndReuploadSection';
import { FinishedTab } from '@/pages/dashboard/assessment/FinishedTab';
import { AssessmentCardSkeleton } from '@/components/skeletons/AssessmentCardSkeleton';
import { GeneralAssessmentCardSkeleton } from '@/components/skeletons/GeneralAssessmentCardSkeleton';
import { RelatedSkilAsessmentsSectionSkeleton } from '@/components/skeletons/RelatedSkilAsessmentsSectionSkeleton';
import { CustomizationAndReuploadSectionSkeleton } from '@/components/skeletons/CustomizationAndReuploadSectionSkeleton';

export const Route = createFileRoute('/_app/app/assessment/')({
  component: RouteComponent,
});

interface Assessment {
  id: string;
  skillName: string;
  difficulty: 'Beginner' | 'Intermediate' | 'Advanced';
  estimatedTime: string;
  questionCount: number;
  isRecommended: boolean;
  category: string;
  progress?: number;
  status?: 'available' | 'in-progress' | 'completed';
  score?: number;
  completedDate?: string;
  passed?: boolean;
  sessionId?: string;
  level?: string;
  expiresAt?: string;
}

interface GeneralAssessment {
  id: string;
  title: string;
  description: string;
  difficulty: 'Beginner' | 'Intermediate' | 'Advanced';
  estimatedTime: string;
  questionCount: number;
  topics: string[];
  isRecommended: boolean;
  status?: string;
  sessionId?: string;
  score?: number;
  level?: string;
}

interface CVAssessmentFromAPI {
  id: string;
  type: 'general' | 'skill' | 'related_skill';
  title: string;
  description?: string;
  difficulty: 'Beginner' | 'Intermediate' | 'Advanced';
  estimatedTime: string;
  questionCount: number;
  topics?: string[];
  isRecommended: boolean;
  category?: string;
  status: string;
  sessionId?: string;
  score?: number;
  level?: string;
  expiresAt?: string;
  progress?: number;
}

function RouteComponent() {
  const [availableAssessments, setAvailableAssessments] = React.useState<
    Assessment[]
  >([]);
  const [relatedAssessments, setRelatedAssessments] = React.useState<
    Assessment[]
  >([]);
  const [inProgressAssessments, setInProgressAssessments] = React.useState<
    Assessment[]
  >([]);
  const [completedAssessments, setCompletedAssessments] = React.useState<
    Assessment[]
  >([]);
  const [generalAssessments, setGeneralAssessments] = React.useState<
    GeneralAssessment[]
  >([]);
  const [roleAssessmentCompleted, setRoleAssessmentCompleted] =
    React.useState(false);
  const [isLoadingProfile, setIsLoadingProfile] = React.useState(true);
  const [profileExists, setProfileExists] = React.useState(true);

  const fetchAssessments = React.useCallback(async () => {
    try {
      const [profileRes, certRes] = await Promise.all([
        fetch('/api/profile', { credentials: 'include' }),
        fetch('/api/certificates/user', { credentials: 'include' }),
      ]);

      let userCertificates: any[] = [];
      if (certRes.ok) {
        const certData = await certRes.json();
        userCertificates = certData.certificates || [];
      }

      if (profileRes.ok) {
        const data = await profileRes.json();
        if (
          data.profile &&
          data.profile.cvAssessments &&
          data.profile.cvAssessments.length > 0
        ) {
          const allAssessments = data.profile
            .cvAssessments as CVAssessmentFromAPI[];

          const gen = allAssessments.filter(
            (a: CVAssessmentFromAPI) =>
              a.type === 'general' &&
              a.status !== 'completed' &&
              a.status !== 'in-progress',
          );
          const availableSkills = allAssessments.filter(
            (a: CVAssessmentFromAPI) =>
              a.type === 'skill' && (a.status === 'available' || !a.status),
          );
          const relatedSkills = allAssessments.filter(
            (a: CVAssessmentFromAPI) =>
              a.type === 'related_skill' &&
              (a.status === 'available' || !a.status),
          );
          const completed = allAssessments.filter(
            (a: CVAssessmentFromAPI) => a.status === 'completed',
          );
          const inProgress = allAssessments.filter(
            (a: CVAssessmentFromAPI) => a.status === 'in-progress',
          );

          setGeneralAssessments(
            gen.map((a: CVAssessmentFromAPI) => ({
              id: a.id,
              title: a.title,
              description:
                a.description ||
                'Menguji kompetensi komprehensif terkait role.',
              difficulty: a.difficulty || 'Intermediate',
              estimatedTime: a.estimatedTime || '90 menit',
              questionCount: a.questionCount || 50,
              topics: a.topics || [],
              isRecommended: a.isRecommended,
              status: a.status,
              sessionId: a.sessionId,
              score: a.score,
              level: a.level,
            })),
          );

          setAvailableAssessments(
            availableSkills.map((a: CVAssessmentFromAPI) => ({
              id: a.id,
              skillName: a.title,
              difficulty: a.difficulty || 'Intermediate',
              estimatedTime: a.estimatedTime || '45 menit',
              questionCount: a.questionCount || 30,
              isRecommended: a.isRecommended,
              category: a.category || 'General',
              status: 'available',
            })),
          );

          setRelatedAssessments(
            relatedSkills.map((a: CVAssessmentFromAPI) => ({
              id: a.id,
              skillName: a.title,
              difficulty: a.difficulty || 'Intermediate',
              estimatedTime: a.estimatedTime || '45 menit',
              questionCount: a.questionCount || 30,
              isRecommended: a.isRecommended,
              category: a.category || 'General',
              status: 'available',
            })),
          );

          setInProgressAssessments(
            inProgress.map((a: CVAssessmentFromAPI) => ({
              id: a.id,
              skillName: a.title,
              difficulty: a.difficulty || 'Intermediate',
              estimatedTime: a.estimatedTime || '45 menit',
              questionCount: a.questionCount || 30,
              isRecommended: a.isRecommended,
              category:
                a.category || (a.type === 'general' ? 'General Role' : 'Skill'),
              status: 'in-progress',
              sessionId: a.sessionId,
              expiresAt: a.expiresAt,
              progress: a.progress,
            })),
          );

          setCompletedAssessments(
            completed.map((a: CVAssessmentFromAPI) => {
              const matchingCerts = userCertificates.filter(
                (cert: any) => cert.assessmentName === a.title,
              );

              let bestScore = a.score || 0;
              let bestLevel = a.level || 'Intermediate';

              if (matchingCerts.length > 0) {
                const sortedCerts = [...matchingCerts].sort(
                  (x, y) => (y.score || 0) - (x.score || 0),
                );
                bestScore = sortedCerts[0].score || bestScore;
                bestLevel = sortedCerts[0].level || bestLevel;
              }

              return {
                id: a.id,
                skillName: a.title,
                difficulty: a.difficulty || 'Intermediate',
                estimatedTime: a.estimatedTime || '45 menit',
                questionCount: a.questionCount || 30,
                isRecommended: a.isRecommended,
                category:
                  a.category || (a.type === 'general' ? 'General' : 'Skill'),
                status: 'completed',
                sessionId: a.sessionId,
                score: bestScore,
                level: bestLevel,
              };
            }),
          );

          const hasUncompletedGeneral = allAssessments.some(
            (a: CVAssessmentFromAPI) =>
              a.type === 'general' && a.status !== 'completed',
          );
          setRoleAssessmentCompleted(
            !!data.profile.roleAssessmentCompleted || !hasUncompletedGeneral,
          );

          setProfileExists(true);
          setIsLoadingProfile(false);
          return;
        }
      }
    } catch (e) {
      console.error('Failed to load profile assessments', e);
    }

    // No mock data if user profile or cvAssessments does not exist in MongoDB
    setAvailableAssessments([]);
    setRelatedAssessments([]);
    setInProgressAssessments([]);
    setCompletedAssessments([]);
    setGeneralAssessments([]);
    setRoleAssessmentCompleted(false);
    setProfileExists(false);
    setIsLoadingProfile(false);
  }, []);

  React.useEffect(() => {
    fetchAssessments();
  }, [fetchAssessments]);

  if (isLoadingProfile) {
    return (
      <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="space-y-6">
          {/* Header Section */}
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Assasemen</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Ikuti assessment untuk mendapatkan kredensial blockchain
            </p>
          </div>

          {/* Tabs */}
          <Tabs defaultValue="available" className="w-full space-y-4">
            <TabsList className="grid w-full grid-cols-2 bg-muted p-1 h-auto">
              <TabsTrigger
                value="available"
                className="flex items-center gap-2"
              >
                <span>Tersedia</span>
              </TabsTrigger>
              <TabsTrigger
                value="completed"
                className="flex items-center gap-2"
              >
                <span>Selesai</span>
              </TabsTrigger>
            </TabsList>

            <TabsContent value="available" className="space-y-3">
              {/* General Assessments Section */}
              <div className="space-y-4">
                <div className="flex items-center justify-between pb-3 border-b">
                  <h3 className="text-lg font-bold">Asesmen Role-based</h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {[...Array(3)].map((_, i) => (
                    <GeneralAssessmentCardSkeleton key={i} />
                  ))}
                </div>
              </div>

              {/* Skill Assessments Section */}
              <div className="space-y-4 pt-4">
                <div className="flex items-center justify-between pb-3 border-b">
                  <h3 className="text-lg font-bold">Asesmen Spesifik Skill</h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {[...Array(4)].map((_, i) => (
                    <AssessmentCardSkeleton key={i} />
                  ))}
                </div>
              </div>

              {/* Related Skill Assessments Section */}
              <RelatedSkilAsessmentsSectionSkeleton />

              {/* Customization & Re-upload Section */}
              <CustomizationAndReuploadSectionSkeleton />
            </TabsContent>
          </Tabs>
        </div>
      </main>
    );
  }

  return (
    <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="space-y-6">
        {/* Header Section */}
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Assasemen</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Ikuti assessment untuk mendapatkan kredensial blockchain
          </p>
        </div>

        {/* Tabs */}
        <Tabs defaultValue="available" className="w-full space-y-4">
          <TabsList
            className={`grid w-full bg-muted p-1 h-auto ${
              inProgressAssessments.length > 0 ? 'grid-cols-3' : 'grid-cols-2'
            }`}
          >
            <TabsTrigger
              value="available"
              className="flex items-center gap-2 transition-all data-[state=active]:bg-background data-[state=active]:shadow-sm hover:bg-background/50"
            >
              <span>Tersedia</span>
              <span className="flex h-5 w-5 items-center justify-center rounded-full border text-xs bg-background data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
                {availableAssessments.length +
                  generalAssessments.length +
                  relatedAssessments.length}
              </span>
            </TabsTrigger>

            {inProgressAssessments.length > 0 && (
              <TabsTrigger
                value="in-progress"
                className="flex items-center gap-2 transition-all data-[state=active]:bg-background data-[state=active]:shadow-sm hover:bg-background/50"
              >
                <span>Berjalan</span>
                <span className="rounded-full border bg-background px-2 py-0.5 text-xs">
                  {inProgressAssessments.length}
                </span>
              </TabsTrigger>
            )}

            <TabsTrigger
              value="completed"
              className="flex items-center gap-2 transition-all data-[state=active]:bg-background data-[state=active]:shadow-sm hover:bg-background/50"
            >
              <span>Selesai</span>
              <span className="rounded-full border bg-background px-2 py-0.5 text-xs">
                {completedAssessments.length}
              </span>
            </TabsTrigger>
          </TabsList>

          {/* Tab: Tersedia */}
          <TabsContent value="available" className="space-y-3">
            {!profileExists ? (
              <Card className="py-12 border">
                <CardContent className="text-center space-y-4">
                  <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto">
                    <svg
                      className="w-8 h-8 text-muted-foreground"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                      />
                    </svg>
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold mb-2">
                      Belum ada data skill
                    </h3>
                    <p className="text-muted-foreground text-sm max-w-sm mx-auto">
                      Silakan unggah CV Anda di halaman Parse CV untuk
                      menganalisis skill dan menampilkan rekomendasi asesmen.
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : (
              <>
                {/* General Assessments Section */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between pb-3 border-b">
                    <h3 className="text-lg font-bold">Asesmen Role-based</h3>
                    <span className="text-sm font-medium text-muted-foreground">
                      {generalAssessments.length} Asesmen
                    </span>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {generalAssessments.map((assessment) => (
                      <GeneralAssessmentCard
                        key={assessment.id}
                        assessment={assessment}
                      />
                    ))}
                  </div>
                </div>

                {/* Skill Assessments Section */}
                <div className="space-y-4 pt-4">
                  <div className="flex items-center justify-between pb-3 border-b">
                    <h3 className="text-lg font-bold">
                      Asesmen Spesifik Skill
                    </h3>
                    <span className="text-sm font-medium text-muted-foreground">
                      {availableAssessments.length} Asesmen
                    </span>
                  </div>
                  {availableAssessments.length > 0 ? (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {availableAssessments.map((assessment) => (
                        <AssessmentCard
                          key={assessment.id}
                          assessment={assessment}
                        />
                      ))}
                    </div>
                  ) : (
                    <Card className="py-12 border">
                      <CardContent className="text-center">
                        <p className="text-muted-foreground">
                          Tidak ada assessment spesifik skill tersedia saat ini.
                        </p>
                      </CardContent>
                    </Card>
                  )}
                </div>

                {/* Related Skill Assessments Section */}
                <RelatedSkilAsessmentsSection
                  relatedAssessments={relatedAssessments}
                  roleAssessmentCompleted={roleAssessmentCompleted}
                />

                {/* Customization & Re-upload Section */}
                <CustomizationAndReuploadSection
                  roleAssessmentCompleted={roleAssessmentCompleted}
                  onRefresh={fetchAssessments}
                />
              </>
            )}
          </TabsContent>

          {/* Tab: Berjalan */}
          {inProgressAssessments.length > 0 && (
            <TabsContent value="in-progress" className="space-y-3">
              {!profileExists ? (
                <Card className="py-12 border">
                  <CardContent className="text-center">
                    <p className="text-muted-foreground text-sm">
                      Silakan unggah CV Anda terlebih dahulu.
                    </p>
                  </CardContent>
                </Card>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {inProgressAssessments.map((assessment) => (
                    <AssessmentCard
                      key={assessment.id}
                      assessment={assessment}
                    />
                  ))}
                </div>
              )}
            </TabsContent>
          )}

          {/* Tab: Selesai */}
          <FinishedTab
            completedAssessments={completedAssessments}
            profileExists={profileExists}
          />
        </Tabs>
      </div>
    </main>
  );
}
