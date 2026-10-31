import type { ResultStatus } from "@/lib/types";

/**
 * ResultStatusBadge — ISSUE-09.
 *
 * The status is always spelled out in words, never carried by colour alone, so
 * it survives a monochrome print, a colour-blind reader and an export. It
 * appears on every results surface, including the certify dialog and history.
 */
const LABELS: Record<ResultStatus, string> = {
  PROVISIONAL: "Provisional",
  CERTIFIED: "Certified",
  SUPERSEDED: "Superseded",
};

const CLASSES: Record<ResultStatus, string> = {
  PROVISIONAL: "rbadge prov",
  CERTIFIED: "rbadge cert",
  SUPERSEDED: "rbadge sup",
};

export function ResultStatusBadge({ status, version }: { status: ResultStatus; version?: number }) {
  return (
    <span className={CLASSES[status]}>
      {LABELS[status]}
      {version ? ` · v${version}` : ""}
    </span>
  );
}
