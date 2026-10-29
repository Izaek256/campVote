"use client";

import { useState } from "react";

interface PrivacyNoticeProps {
  onAcknowledge?: () => void;
  showAcknowledge?: boolean;
}

export function PrivacyNotice({ onAcknowledge, showAcknowledge = false }: PrivacyNoticeProps) {
  const [acknowledged, setAcknowledged] = useState(false);

  const handleAcknowledge = () => {
    setAcknowledged(true);
    if (onAcknowledge) {
      onAcknowledge();
    }
  };

  return (
    <div className="bg-blue-50 border border-blue-200 text-blue-900 px-4 py-3 rounded mb-4">
      <div className="flex items-start gap-3">
        <svg className="w-5 h-5 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clipRule="evenodd"
          />
        </svg>
        <div className="flex-1">
          <p className="font-medium text-sm">Your Ballot Privacy</p>
          <p className="text-xs mt-1">
            Your choices are stored anonymously. Election officers can see turnout counts but not individual choices.{" "}
            <a href="/privacy/ballot-secrecy" className="underline hover:text-blue-700">
              Read full privacy statement
            </a>
          </p>
          {showAcknowledge && !acknowledged && (
            <button
              onClick={handleAcknowledge}
              className="mt-2 text-xs bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700"
            >
              I understand
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
