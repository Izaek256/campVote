import type { LandingHero } from "@campvote/shared/types/index";
import { HeroArt } from "./landing-art";

const CHIPS = ["Secret ballot", "Roster-checked", "Mobile-first", "Audit trail"];

/** Colour the one word the editor wrapped in asterisks; print the rest as-is. */
function Headline({ text }: { text: string }) {
  return (
    <>
      {text.split("\n").map((line, lineIndex) => (
        // Lines are content, not layout, so the index is a stable key here.
        <span key={lineIndex}>
          {lineIndex > 0 ? <br /> : null}
          {line
            .split(/\*([^*]+)\*/g)
            .map((part, index) => (index % 2 === 1 ? <em key={index}>{part}</em> : part))}
        </span>
      ))}
    </>
  );
}

/** Hero copy, the ballot illustration, and the two entry points. */
export function LandingHeroSection({ hero }: { hero: LandingHero }) {
  return (
    <header className="lp-hero">
      <div className="lp-w">
        <div className="lp-hg">
          <div className="lp-rv">
            <div className="lp-mo">Campus elections · one voter, one ballot</div>
            <h1 style={{ marginTop: 14 }}>
              <Headline text={hero.headline} />
            </h1>
            <p className="lp-lead">{hero.subtext}</p>
            <div className="lp-ac">
              <a className="lp-bt" href="/login">
                {hero.primary_label} →
              </a>
              <a className="lp-bt lp-o" href="#how">
                {hero.secondary_label}
              </a>
            </div>
            <div className="lp-ch">
              {CHIPS.map((chip) => (
                <span key={chip}>{chip}</span>
              ))}
            </div>
          </div>
          <div className="lp-sc lp-rv" style={{ "--d": ".15s" } as React.CSSProperties}>
            <HeroArt />
          </div>
        </div>
      </div>
    </header>
  );
}
