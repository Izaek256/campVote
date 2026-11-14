import { SecurityArt } from "./landing-art";

const CARDS = [
  {
    title: "Identity and choice never meet",
    text: "We record that you voted, and drop the ballot into the box with no name on it. No table links the two.",
  },
  {
    title: "One voter, one ballot",
    text: "Eligibility and duplicates are enforced on the server, even if two taps arrive at the same moment.",
  },
  {
    title: "A tamper-evident audit trail",
    text: "Admin actions and ballot snapshots form a tamper-evident audit trail, so changes after the fact are visible to auditors.",
  },
];

const TAGS = [
  "Server-side eligibility",
  "Role-based access",
  "Config locked once voting opens",
  "Aggregate-only observer view",
  "Activity and audit logs",
];

/**
 * Security claims are fixed in code on purpose: the CMS cannot edit or remove
 * a statement that describes how the system protects a ballot.
 */
export function LandingSecuritySection() {
  return (
    <section className="lp-sec lp-band lp-sx" id="security">
      <div className="lp-w">
        <div className="lp-mo lp-rv">Security and privacy</div>
        <h2 className="lp-h2 lp-rv">Your vote is secret. The process is not.</h2>
        <p className="lp-sub lp-rv">
          The system proves that you voted and that every ballot counts, without ever tying your
          name to your choice.
        </p>

        <div className="lp-sg">
          {CARDS.map((card, index) => (
            <div
              className="lp-cd lp-rv"
              key={card.title}
              style={{ "--d": `${index * 0.12}s` } as React.CSSProperties}
            >
              <SecurityArt index={index} />
              <div>
                <h3>{card.title}</h3>
                <p>{card.text}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="lp-tg lp-rv">
          {TAGS.map((tag) => (
            <span key={tag}>{tag}</span>
          ))}
        </div>
      </div>
    </section>
  );
}
