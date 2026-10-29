import type { Ballot, BallotPosition } from "./types";

export const ABSTAIN = "ABSTAIN";

export function blankAllowed(position: BallotPosition): boolean {
  return position.allow_blank || position.min_selections === 0;
}

export function isReferendum(position: BallotPosition): boolean {
  return position.question_type === "REFERENDUM";
}

export function limits(position: BallotPosition): { min: number; max: number } {
  const max = position.question_type === "MULTI_CHOICE" ? Math.max(1, position.max_selections) : 1;
  return { min: Math.max(0, position.min_selections), max };
}

/** Mirrors the server's rules for a first pass; the server is still the gate. */
export function validate(
  ballot: Ballot,
  selections: Record<string, string[]>
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const position of ballot.positions) {
    const chosen = selections[position.id] ?? [];
    const { min, max } = limits(position);
    if (chosen.includes(ABSTAIN) && chosen.length > 1) {
      errors[position.id] = "An abstention must be the only choice for this position.";
      continue;
    }
    if (chosen.includes(ABSTAIN) && !blankAllowed(position)) {
      errors[position.id] = "This position cannot be left blank.";
      continue;
    }
    if (chosen.length < min) {
      errors[position.id] =
        chosen.length === 0
          ? "Choose an option for this position."
          : `Choose ${min - chosen.length} more option${min - chosen.length === 1 ? "" : "s"}.`;
      continue;
    }
    if (chosen.length > max) {
      errors[position.id] = `Choose at most ${max} option${max === 1 ? "" : "s"}.`;
    }
  }
  return errors;
}

export function buildSelections(
  ballot: Ballot,
  selections: Record<string, string[]>
): Array<Record<string, unknown>> {
  const rows: Array<Record<string, unknown>> = [];
  for (const position of ballot.positions) {
    for (const value of selections[position.id] ?? []) {
      if (value === ABSTAIN || value === "YES" || value === "NO") {
        rows.push({ position_id: position.id, choice: value });
      } else {
        rows.push({ position_id: position.id, candidate_id: value });
      }
    }
  }
  return rows;
}
