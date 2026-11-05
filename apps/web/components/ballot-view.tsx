"use client";

import type { BallotContent } from "@/lib/types";
import { Icon } from "./icon";

interface BallotViewProps {
  content: BallotContent;
  mode: "preview" | "vote";
  onSelectionChange?: (positionId: string, selections: string[]) => void;
  selections?: Record<string, string[]>;
}

export function BallotView({ content, mode, onSelectionChange, selections = {} }: BallotViewProps) {
  const isPreview = mode === "preview";

  const handleSelection = (positionId: string, candidateId: string, questionType: string) => {
    if (isPreview || !onSelectionChange) return;

    const current = selections[positionId] || [];
    let next: string[];

    if (questionType === "SINGLE_CHOICE" || questionType === "YES_NO") {
      next = [candidateId];
    } else if (current.includes(candidateId)) {
      next = current.filter((id) => id !== candidateId);
    } else {
      next = [...current, candidateId];
    }

    onSelectionChange(positionId, next);
  };

  const handleAbstain = (positionId: string) => {
    if (isPreview || !onSelectionChange) return;
    onSelectionChange(positionId, ["ABSTAIN"]);
  };

  return (
    <div className="stack">
      {content.positions.map((position) => {
        const selected = selections[position.id] || [];
        const count = selected.length;
        const isFull = count >= position.max_selections;
        const isAbstained = selected.includes("ABSTAIN");
        const multi = position.question_type === "MULTI_CHOICE";
        const blocked = isPreview;

        return (
          <section className="ballot-position" key={position.id}>
            <div className="row-between">
              <div>
                <span className="lbl">
                  {position.min_selections > 0 ? "Required" : "Optional"}
                  {position.seats && position.seats > 1 ? ` · ${position.seats} seats` : ""}
                </span>
                <h2 style={{ fontSize: 24, marginTop: 4 }}>{position.title}</h2>
                {position.description ? <p className="hint">{position.description}</p> : null}
              </div>
              {multi ? (
                <span className={`counter${count === position.max_selections ? " full" : ""}`}>
                  {count} of {position.max_selections} selected
                </span>
              ) : null}
            </div>

            {/* biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is group or radiogroup; both support aria-label. */}
            <div
              className="ballot-list"
              role={multi ? "group" : "radiogroup"}
              aria-label={position.title}
            >
              {position.candidates.map((candidate) => {
                const isSelected = selected.includes(candidate.id);
                const isDisabled = blocked || (isFull && !isSelected) || isAbstained;
                return (
                  // biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is a literal radio or checkbox; both support aria-checked.
                  <button
                    type="button"
                    className="opt"
                    key={candidate.id}
                    role={multi ? "checkbox" : "radio"}
                    aria-checked={isSelected}
                    aria-disabled={isDisabled}
                    disabled={isDisabled}
                    onClick={() =>
                      handleSelection(position.id, candidate.id, position.question_type)
                    }
                  >
                    <span className={`box${multi ? "" : " rd"}`}>
                      {multi ? <Icon name="check" size="s" /> : null}
                    </span>
                    <span className="ph r11" style={{ width: 48, flex: "none" }} aria-hidden="true">
                      <span style={{ padding: "0 4px" }}>{candidate.ballot_number}</span>
                    </span>
                    <span style={{ minWidth: 0 }}>
                      <span className="nm">{candidate.full_name}</span>
                      {candidate.manifesto ? (
                        <span className="candidate-meta">{candidate.manifesto}</span>
                      ) : null}
                    </span>
                  </button>
                );
              })}

              {position.allow_blank ? (
                // biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is a literal radio or checkbox; both support aria-checked.
                <button
                  type="button"
                  className="opt"
                  role={multi ? "checkbox" : "radio"}
                  aria-checked={isAbstained}
                  aria-disabled={blocked}
                  disabled={blocked}
                  onClick={() => handleAbstain(position.id)}
                >
                  <span className={`box${multi ? "" : " rd"}`}>
                    {multi ? <Icon name="check" size="s" /> : null}
                  </span>
                  <span className="nm">Leave this position blank</span>
                </button>
              ) : null}
            </div>

            {!isPreview && count < position.min_selections ? (
              <span className="err" style={{ display: "flex", gap: 6, marginTop: 10 }}>
                <Icon name="alert" size="s" />
                Choose at least {position.min_selections} option
                {position.min_selections > 1 ? "s" : ""}.
              </span>
            ) : null}
          </section>
        );
      })}
    </div>
  );
}
