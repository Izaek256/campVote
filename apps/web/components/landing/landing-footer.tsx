import type { LandingFooter } from "@campvote/shared/types/index";
import { mailto } from "@/lib/landing";
import { LogoMark } from "./landing-art";

/** Closing line, support address when configured, and the staff door. */
export function LandingFooterSection({ footer }: { footer: LandingFooter }) {
  const support = mailto(footer.support_email);
  return (
    <footer className="lp-footer">
      <div className="lp-w">
        <a className="lp-lg" href="#top">
          <span className="lp-mk">
            <LogoMark />
          </span>
          <span>
            camp<b>Vote</b>
          </span>
        </a>
        <span>
          {footer.text}
          {support ? (
            <>
              {" "}
              <a href={support}>{footer.support_email}</a>
            </>
          ) : null}
        </span>
        <a href="/login">Staff sign in</a>
      </div>
    </footer>
  );
}
