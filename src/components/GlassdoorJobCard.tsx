import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Briefcase, Clock, MapPin } from 'lucide-react';
import type { Job } from '@/lib/jobs-client';
import GlassdoorIcon from '@/assets/svg/GlassDoor.svg';

interface GlassdoorJobCardProps {
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

function formatSalary(salary: string): string {
  try {
    // Format: "USD 100000 - 215000 (ANNUAL)"
    const match = salary.match(/^(\w+)\s+([\d.]+)\s*-\s*([\d.]+)\s*\((\w+)\)$/);
    if (!match) return salary;

    const [, currency, min, max, period] = match;

    // Format numbers with thousand separators
    const formatNumber = (num: string) => {
      return parseFloat(num).toLocaleString('id-ID');
    };

    // Translate period to Indonesian
    const periodMap: Record<string, string> = {
      ANNUAL: 'Per Tahun',
      MONTHLY: 'Per Bulan',
      HOURLY: 'Per Jam',
      WEEKLY: 'Per Minggu',
    };

    const periodIndo = periodMap[period] || period;

    return `${currency} ${formatNumber(min)} - ${formatNumber(max)} (${periodIndo})`;
  } catch {
    return salary;
  }
}

function stripHtml(html: string): string {
  try {
    // Remove HTML tags
    let text = html.replace(/<[^>]*>/g, '');

    // Decode common HTML entities
    text = text
      .replace(/&nbsp;/g, ' ')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/&mdash;/g, '—')
      .replace(/&ndash;/g, '–');

    // Remove extra whitespace
    text = text.replace(/\s+/g, ' ').trim();

    return text;
  } catch {
    return html;
  }
}

export function GlassdoorJobCard({ job }: GlassdoorJobCardProps) {
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
            src={job.logo || GlassdoorIcon}
            alt={job.logo ? job.company : 'Glassdoor'}
            className="h-full w-full object-contain"
            onError={(e) => {
              e.currentTarget.src = GlassdoorIcon;
              e.currentTarget.alt = 'Glassdoor';
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
                {job.companyUrl ? (
                  <a
                    href={job.companyUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium hover:text-primary"
                  >
                    {job.company}
                  </a>
                ) : (
                  <span className="font-medium">{job.company}</span>
                )}
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
                <span>{formatSalary(job.salary)}</span>
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
            <div className="mt-4 flex flex-wrap gap-2">
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
              {stripHtml(job.description)}
            </p>
          )}
        </div>
      </div>
    </article>
  );
}
