/** The looping "one voter · one ballot" strip. Decorative, so it is hidden from AT. */
export function LandingTicker({ phrases }: { phrases: string[] }) {
  if (phrases.length === 0) return null;
  const text = phrases.join(" · ");
  // Two copies make the marquee's translateX(-50%) loop seamless.
  return (
    <div className="lp-tk" aria-hidden="true">
      <div>
        {text} · {text} ·
      </div>
    </div>
  );
}
