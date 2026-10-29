"use client";

import { BallotContent } from "@campvote/shared/types/ballot-version";

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
    let newSelections: string[];

    if (questionType === "SINGLE_CHOICE" || questionType === "YES_NO") {
      newSelections = [candidateId];
    } else {
      // Multi-choice
      if (current.includes(candidateId)) {
        newSelections = current.filter((id) => id !== candidateId);
      } else {
        newSelections = [...current, candidateId];
      }
    }

    onSelectionChange(positionId, newSelections);
  };

  const handleAbstain = (positionId: string) => {
    if (isPreview || !onSelectionChange) return;
    onSelectionChange(positionId, ["ABSTAIN"]);
  };

  const getSelectionCount = (positionId: string) => {
    return selections[positionId]?.length || 0;
  };

  return (
    <div className="space-y-8">
      {isPreview && (
        <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 px-4 py-3 rounded">
          <p className="font-medium">Preview Mode</p>
          <p className="text-sm">No vote or participation is recorded. This is a ballot preview only.</p>
        </div>
      )}

      {content.positions.map((position) => {
        const selected = selections[position.id] || [];
        const count = selected.length;
        const isFull = count >= position.max_selections;
        const isAbstained = selected.includes("ABSTAIN");

        return (
          <div key={position.id} className="border rounded-lg p-6">
            <div className="mb-4">
              <h3 className="text-lg font-semibold">{position.title}</h3>
              {position.description && (
                <p className="text-gray-600 mt-1">{position.description}</p>
              )}
              <div className="mt-2 flex items-center gap-4 text-sm text-gray-500">
                <span>Seats: {position.seats}</span>
                <span>
                  Selections: {count} / {position.max_selections}
                  {position.min_selections > 0 && ` (min: ${position.min_selections})`}
                </span>
              </div>
            </div>

            <div className="space-y-3">
              {position.candidates.map((candidate) => {
                const isSelected = selected.includes(candidate.id);
                const isDisabled = isPreview || (isFull && !isSelected) || isAbstained;

                return (
                  <button
                    key={candidate.id}
                    onClick={() => handleSelection(position.id, candidate.id, position.question_type)}
                    disabled={isDisabled}
                    className={`w-full text-left p-4 border-2 rounded-lg transition-all ${
                      isSelected
                        ? "border-green-600 bg-green-50"
                        : "border-gray-200 hover:border-gray-300"
                    } ${isDisabled ? "opacity-50 cursor-not-allowed" : ""}`}
                    aria-pressed={isSelected}
                    aria-disabled={isDisabled}
                  >
                    <div className="flex items-start gap-4">
                      <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gray-200 flex items-center justify-center text-gray-600 font-medium">
                        {candidate.ballot_number}
                      </div>
                      <div className="flex-1">
                        <div className="font-medium">{candidate.full_name}</div>
                        {candidate.manifesto && (
                          <p className="text-sm text-gray-600 mt-1 line-clamp-2">
                            {candidate.manifesto}
                          </p>
                        )}
                      </div>
                      {isSelected && (
                        <div className="flex-shrink-0 text-green-600">
                          <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                        </div>
                      )}
                    </div>
                  </button>
                );
              })}

              {position.allow_blank && !isPreview && (
                <button
                  onClick={() => handleAbstain(position.id)}
                  disabled={isAbstained}
                  className={`w-full text-left p-4 border-2 rounded-lg transition-all ${
                    isAbstained
                      ? "border-gray-400 bg-gray-100"
                      : "border-gray-200 hover:border-gray-300"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <span className="font-medium">Abstain</span>
                    {isAbstained && (
                      <svg className="w-5 h-5 text-gray-600" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fillRule="evenodd"
                          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                          clipRule="evenodd"
                        />
                      </svg>
                    )}
                  </div>
                </button>
              )}
            </div>

            {count < position.min_selections && !isPreview && (
              <p className="mt-3 text-sm text-red-600">
                Minimum {position.min_selections} selection{position.min_selections > 1 ? "s" : ""} required
              </p>
            )}
          </div>
        );
      })}
    </div>
  );
}
