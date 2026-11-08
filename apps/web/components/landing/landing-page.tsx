import type { LandingDocument, PublicUpcomingElection } from "@campvote/shared/types/index";
import { bannerVisible } from "@/lib/landing";
import { LandingCtaSection } from "./landing-cta";
import { LandingEffects } from "./landing-effects";
import { LandingFaqSection } from "./landing-faq";
import { LandingFeaturesSection } from "./landing-features";
import { LandingFooterSection } from "./landing-footer";
import { LandingHeroSection } from "./landing-hero";
import { LandingHowSection } from "./landing-how";
import { LandingNav } from "./landing-nav";
import { LandingSecuritySection } from "./landing-security";
import { LandingTicker } from "./landing-ticker";
import { LandingUpcoming } from "./landing-upcoming";

/**
 * The whole public landing page. The admin preview renders the same component
 * with its draft and no live election list, so what an administrator previews
 * is what publish will produce.
 */
export function LandingPage({
  document,
  upcoming,
  upcomingFailed,
  preview = false,
}: {
  document: LandingDocument;
  upcoming: PublicUpcomingElection[];
  upcomingFailed: boolean;
  preview?: boolean;
}) {
  return (
    <div
      className={`landing${preview ? " lp-preview" : ""}${bannerVisible(document.banner) ? "" : " lp-nobn"}`}
    >
      <noscript
        dangerouslySetInnerHTML={{
          __html: "<style>.lp-rv,.lp-sx{opacity:1!important;transform:none!important}</style>",
        }}
      />
      <a className="lp-skip" href="#main">
        Skip to content
      </a>
      {preview ? null : (
        <>
          <LandingNav banner={document.banner} />
          <LandingEffects />
        </>
      )}
      <main id={preview ? undefined : "main"}>
        <div id="top" />
        <LandingHeroSection hero={document.hero} />
        <LandingTicker phrases={document.ticker} />
        <LandingUpcoming initial={upcoming} initialFailed={upcomingFailed} />
        <LandingHowSection how={document.how} />
        <LandingSecuritySection />
        <LandingFeaturesSection features={document.features} />
        <LandingFaqSection faqs={document.faqs} />
        <LandingCtaSection />
      </main>
      <LandingFooterSection footer={document.footer} />
    </div>
  );
}
