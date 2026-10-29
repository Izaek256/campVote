"use client";

interface VoteRecordedNoticeProps {
  votedAt?: string;
  electionTitle?: string;
}

export function VoteRecordedNotice({ votedAt, electionTitle }: VoteRecordedNoticeProps) {
  const formattedTime = votedAt
    ? new Date(votedAt).toLocaleString("en-US", {
        year: "numeric",
        month: "long",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        timeZoneName: "short",
      })
    : "recently";

  return (
    <div className="bg-green-50 border border-green-200 text-green-900 px-6 py-8 rounded-lg text-center">
      <div className="mb-4">
        <svg className="w-16 h-16 mx-auto text-green-600" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
      </div>
      <h2 className="text-2xl font-bold mb-2">Your Vote Has Been Recorded</h2>
      <p className="text-gray-700 mb-4">
        {electionTitle && <span className="font-medium">{electionTitle}</span>}
      </p>
      <p className="text-sm text-gray-600 mb-6">
        Your vote was recorded on {formattedTime}.
      </p>
      <div className="bg-green-100 border border-green-300 rounded-lg p-4 text-sm text-left max-w-md mx-auto">
        <p className="font-medium mb-2">What happens next:</p>
        <ul className="list-disc list-inside space-y-1 text-gray-700">
          <li>Your ballot choices are stored anonymously</li>
          <li>Election officers can see turnout counts but not individual choices</li>
          <li>Results will be published according to the election's result policy</li>
          <li>You can check results when they become available</li>
        </ul>
      </div>
      <div className="mt-6 text-xs text-gray-500">
        <p>
          This confirmation does not reveal or encode your ballot choices. No "download my ballot"
          or "print my vote" feature is provided to protect your privacy.
        </p>
      </div>
    </div>
  );
}
