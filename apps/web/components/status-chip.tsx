import { statusClass, statusLabel } from "@/lib/format";
import type { ElectionStatus } from "@/lib/types";

export function StatusChip({ status }: { status: ElectionStatus | string }) {
  const value = status as ElectionStatus;
  return <span className={`chip ${statusClass(value)}`}>{statusLabel(value)}</span>;
}
