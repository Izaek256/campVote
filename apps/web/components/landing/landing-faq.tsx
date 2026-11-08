import type { LandingFAQ } from "@campvote/shared/types/index";

/** The question list. The first item is open, as in the prototype. */
export function LandingFaqSection({ faqs }: { faqs: LandingFAQ[] }) {
  if (faqs.length === 0) return null;
  return (
    <section className="lp-sec lp-sx" id="faq" style={{ paddingTop: 24 }}>
      <div className="lp-w" style={{ maxWidth: 860 }}>
        <div className="lp-mo lp-rv">FAQ</div>
        <h2 className="lp-h2 lp-rv">Good questions.</h2>
        <div className="lp-rv">
          {faqs.map((faq, index) => (
            <details key={`${faq.question}-${index}`} open={index === 0}>
              <summary>{faq.question}</summary>
              <p>{faq.answer}</p>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}
