import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Briefcase, Clock, MapPin } from 'lucide-react';
import type { Job } from '@/lib/jobs-client';
import UpworkIcon from '@/assets/svg/upwork.svg';

interface UpworkJobCardProps {
  job: Job;
}

function formatDate(dateString: string): string {
  try {
    const date = new Date(dateString);
    const now = new Date();

    const diffDays = Math.floor(
      (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24),
    );

    if (diffDays <= 0) return 'Hari ini';
    if (diffDays === 1) return 'Kemarin';
    if (diffDays < 7) return `${diffDays} hari lalu`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} minggu lalu`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)} bulan lalu`;

    return date.toLocaleDateString('id-ID');
  } catch {
    return dateString;
  }
}

export function UpworkJobCard({ job }: UpworkJobCardProps) {
  // Don't render if essential fields are missing
  if (!job.title || !job.company) {
    return null;
  }

  return (
    <article className="border-b border-border transition-colors hover:bg-muted/30">
      <div className="flex gap-4 px-6 py-5">
        {/* Logo */}
        <div className="flex h-14 w-14 shrink-0 items-center justify-center">
          <img
            src={job.logo || UpworkIcon}
            alt={job.logo ? job.company : 'Upwork'}
            className="h-full w-full object-contain"
            onError={(e) => {
              e.currentTarget.src = UpworkIcon;
              e.currentTarget.alt = 'Upwork';
            }}
          />
        </div>

        {/* Content */}
        <div className="min-w-0 flex-1">
          {/* Title */}
          <div className="flex items-start justify-between gap-6">
            <div className="min-w-0 flex-1">
              {job.url ? (
                <a
                  href={job.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="group inline-flex max-w-full items-center gap-2"
                >
                  <h3 className="truncate text-lg font-semibold transition-colors group-hover:text-primary">
                    {job.title}
                  </h3>
                </a>
              ) : (
                <h3 className="truncate text-lg font-semibold">{job.title}</h3>
              )}

              {/* Company */}
              <div className="mt-1 flex flex-wrap items-center gap-2 text-sm">
                <span className="font-medium">{job.company}</span>
              </div>
            </div>

            {(job.promoted || job.earlyApplicant) && (
              <div className="flex shrink-0 gap-2">
                {job.promoted && (
                  <Badge
                    variant="outline"
                    className="h-6 px-2 text-[11px] font-normal"
                  >
                    Promoted
                  </Badge>
                )}

                {job.earlyApplicant && (
                  <Badge
                    variant="outline"
                    className="h-6 px-2 text-[11px] font-normal"
                  >
                    Early Applicant
                  </Badge>
                )}
              </div>
            )}
          </div>

          {/* Meta */}
          <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <MapPin className="h-3.5 w-3.5" />
              <span>{job.location}</span>
            </div>

            {job.experienceLevel &&
              job.experienceLevel !== 'Not Applicable' && (
                <>
                  <Separator orientation="vertical" className="h-4" />

                  <div className="flex items-center gap-1.5">
                    <Briefcase className="h-3.5 w-3.5" />
                    <span>{job.experienceLevel}</span>
                  </div>
                </>
              )}

            {job.salary && (
              <>
                <Separator orientation="vertical" className="h-4" />
                <span className="font-medium">{job.salary}</span>
              </>
            )}

            {(job.postedDate || job.postedTime) && (
              <>
                <Separator orientation="vertical" className="h-4" />

                <div className="flex items-center gap-1.5">
                  <Clock className="h-3.5 w-3.5" />

                  <span>
                    {job.postedDate
                      ? formatDate(job.postedDate)
                      : job.postedTime}
                  </span>
                </div>
              </>
            )}
          </div>

          {/* Tags */}
          {(job.type || job.sector) && (
            <div className="mt-4 flex flex-wrap gap-2 capitalize">
              {job.type && <Badge variant="default">{job.type}</Badge>}

              {job.sector &&
                job.sector.split(',').map((skill, index) => (
                  <Badge key={index} variant="secondary">
                    {skill.trim()}
                  </Badge>
                ))}
            </div>
          )}

          {/* Description */}

          {job.description && (
            <p className="mt-2 line-clamp-2 text-sm leading-6 text-muted-foreground">
              {job.description}
            </p>
          )}
        </div>
      </div>
    </article>
  );
}
