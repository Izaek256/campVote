export default function BallotSecrecyPage() {
  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-6">Ballot Privacy</h1>

      <div className="prose prose-lg max-w-none">
        <h2 className="text-xl font-semibold mt-6 mb-3">Your Ballot Privacy</h2>
        <p>campVote is designed to protect your ballot choices. Here's what you should know:</p>

        <h3 className="text-lg font-semibold mt-6 mb-2">What is protected:</h3>
        <ul className="list-disc pl-6 space-y-1">
          <li>The specific candidates or options you select are stored anonymously</li>
          <li>Election officers can see turnout counts but cannot see individual choices</li>
          <li>Candidates never see who voted for them</li>
          <li>Your confirmation receipt only shows that you voted, not what you voted for</li>
        </ul>

        <h3 className="text-lg font-semibold mt-6 mb-2">What is not guaranteed:</h3>
        <ul className="list-disc pl-6 space-y-1">
          <li>This is an academic prototype, not a certified election system</li>
          <li>In very small elections, turnout counts alone might reveal who participated</li>
          <li>
            Database administrators with full access have technical capability to access data,
            though this is restricted by policy and audit
          </li>
        </ul>

        <h3 className="text-lg font-semibold mt-6 mb-2">What we do to protect you:</h3>
        <ul className="list-disc pl-6 space-y-1">
          <li>Ballot choices are stored separately from who voted</li>
          <li>
            No identifying information (email, student number, IP address) is stored with your
            choices
          </li>
          <li>Your selections are never saved in your browser or on your device</li>
          <li>All administrative access is logged for audit</li>
        </ul>

        <div className="bg-yellow-50 border border-yellow-200 p-4 rounded-lg mt-8">
          <p className="font-medium">
            If you have concerns about privacy, contact the election administrator or the MMU
            Electoral Commission.
          </p>
        </div>
      </div>
    </div>
  );
}
