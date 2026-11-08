import type { LandingFeature } from "@campvote/shared/types/index";
import { FeatureArt } from "./landing-art";

/** The eight-card feature grid; order, icon, title and text are editable. */
export function LandingFeaturesSection({ features }: { features: LandingFeature[] }) {
  return (
    <section className="lp-sec lp-sx" id="features">
      <div className="lp-w">
        <div className="lp-mo lp-rv">Everything an election needs</div>
        <h2 className="lp-h2 lp-rv">Built for the whole election cycle.</h2>
        <p className="lp-sub lp-rv">
          From the first draft to the certified PDF, administrators, candidates and voters each get
          what they need.
        </p>

        <div className="lp-fg">
          {features.map((feature, index) => (
            <div
              className="lp-ft lp-rv"
              key={`${feature.icon}-${feature.title}`}
              style={{ "--d": `${(index % 4) * 0.08}s` } as React.CSSProperties}
            >
              <FeatureArt icon={feature.icon} />
              <h3>{feature.title}</h3>
              <p>{feature.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
