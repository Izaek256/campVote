"use client";

import { Icon } from "@/components/icon";

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
    : "just now";

  return (
    <section className="panel stack" aria-labelledby="vote-recorded-heading">
      <div className="banner ok">
        <Icon name="check" />
        <div>
          <b id="vote-recorded-heading">Your vote has been recorded</b>
          {electionTitle ? <span>{electionTitle}</span> : null} Recorded {formattedTime}.
        </div>
      </div>
      <p className="hint">
        Your choices are stored separately from the record that you voted. Officials and candidates
        see turnout counts, never individual choices. Results follow the election&apos;s result
        policy.
      </p>
      <p className="hint">
        This confirmation does not reveal or encode your ballot choices. There is no &ldquo;download
        my ballot&rdquo; or &ldquo;print my vote&rdquo; feature.
      </p>
    </section>
  );
}
