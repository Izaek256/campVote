"use client";

import { PrivacyNotice } from "./privacy-notice";

interface PreVoteNoticeProps {
  onContinue?: () => void;
}

export function PreVoteNotice({ onContinue }: PreVoteNoticeProps) {
  return (
    <div className="space-y-4">
      <PrivacyNotice showAcknowledge={false} />
      <div className="bg-yellow-50 border border-yellow-200 text-yellow-900 px-4 py-3 rounded">
        <div className="flex items-start gap-3">
          <svg className="w-5 h-5 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <div className="flex-1">
            <p className="font-medium text-sm">Before You Vote</p>
            <p className="text-xs mt-1">
              Once you submit your ballot, you cannot change it. Make sure you are satisfied with your selections before continuing.
            </p>
          </div>
        </div>
      </div>
      {onContinue && (
        <button
          onClick={onContinue}
          className="w-full px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 font-medium"
        >
          Continue to Ballot
        </button>
      )}
    </div>
  );
}
