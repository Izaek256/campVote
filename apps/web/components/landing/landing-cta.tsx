/** The closing green band with the two ways into the product. */
export function LandingCtaSection() {
  return (
    <section className="lp-cta lp-sx">
      <div className="lp-w">
        <h2 className="lp-rv">Ready to cast your vote?</h2>
        <p className="lp-rv">
          Sign in with your campus account to see the elections you are eligible for.
        </p>
        <div className="lp-rv">
          <a className="lp-bt lp-i" href="/login">
            Sign in with campus Google →
          </a>
          <a className="lp-tl" href="/login">
            Staff sign in
          </a>
        </div>
      </div>
    </section>
  );
}
