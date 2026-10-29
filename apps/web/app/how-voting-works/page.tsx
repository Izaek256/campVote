import Link from "next/link";
import { Icon } from "@/components/icon";

export const metadata = { title: "How voting works — campVote" };

export default function HowVotingWorksPage() {
  return (
    <main className="app-wrap narrow">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <span>How voting works</span>
      </nav>
      <div className="page-head">
        <div>
          <span className="lbl">Guide</span>
          <h1>How voting works</h1>
          <p>A short, honest guide to the campVote academic prototype.</p>
        </div>
      </div>

      <div className="stack">
        <section className="panel stack">
          <h2 style={{ fontSize: 24 }}>1. Sign in with your MMU account</h2>
          <p>
            Use your verified <span className="mono">@std.must.ac.ug</span> or{" "}
            <span className="mono">@must.ac.ug</span> address. Holding an institutional address is
            only an identity hint: the server also checks the election roster.
          </p>
        </section>
        <section className="panel stack">
          <h2 style={{ fontSize: 24 }}>2. Check your eligibility</h2>
          <p>
            Every ballot request runs the same server-side roster check. A hidden or disabled button
            is never the only control.
          </p>
        </section>
        <section className="panel stack">
          <h2 style={{ fontSize: 24 }}>3. Fill in the ballot</h2>
          <p>
            Required positions must be answered. Optional positions may be left blank. The counter
            shows how many choices a multi-choice position allows.
          </p>
        </section>
        <section className="panel stack">
          <h2 style={{ fontSize: 24 }}>4. Submit once</h2>
          <p>
            A ballot is final once accepted. You cannot change or submit it again, and the server
            enforces one accepted ballot per voter per election.
          </p>
        </section>
        <div className="banner">
          <Icon name="lock" />
          <div>
            <b>What stays secret</b>
            Participation is recorded separately from ballot choices. Officials, candidates and the
            public see totals, never a voter-to-choice mapping. This prototype is not certified for
            legally binding elections.
          </div>
        </div>
        <p>
          <Link className="btn primary" href="/elections">
            Back to my elections
          </Link>
        </p>
      </div>
    </main>
  );
}
