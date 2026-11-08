import type { LandingHow } from "@campvote/shared/types/index";
import { StepArt } from "./landing-art";

const LIFECYCLE = ["Draft", "Registration", "Scheduled", "Open", "Closed", "Review", "Certified"];

/** The four (or more) steps, the lifecycle strip and the results-visibility note. */
export function LandingHowSection({ how }: { how: LandingHow }) {
  return (
    <section className="lp-sec lp-sx" id="how">
      <div className="lp-w">
        <div className="lp-mo lp-rv">How voting works</div>
        <h2 className="lp-h2 lp-rv">Four steps, about two minutes.</h2>
        <p className="lp-sub lp-rv">
          No passwords to remember and nothing to install. Sign in with your campus account and vote
          from your phone.
        </p>

        <div className={`lp-st lp-rv${how.steps.length > 4 ? " lp-st-many" : ""}`}>
          {how.steps.map((step, index) => (
            <div className="lp-stp" key={`${step.title}-${index}`}>
              <div className="lp-n">{String(index + 1).padStart(2, "0")}</div>
              <StepArt index={index} />
              <h3>{step.title}</h3>
              <p>{step.text}</p>
            </div>
          ))}
        </div>

        <ol className="lp-lc lp-rv" aria-label="Election lifecycle">
          {LIFECYCLE.map((label, index) => (
            <li key={label} style={{ "--i": index } as React.CSSProperties}>
              {label}
            </li>
          ))}
        </ol>
        <div className="lp-nt lp-rv">
          <b>Results stay hidden while voting is open.</b>
          Live counts are configurable per election, so early leaders cannot sway people who have
          not voted yet.
        </div>
      </div>
    </section>
  );
}
