import type { Metadata } from "next";
import { LandingPage } from "@/components/landing/landing-page";
import { LandingSignedInRedirect } from "@/components/landing/landing-redirect";
import { CAMPUS_NAME, SITE_URL } from "@/lib/config";
import { fetchLanding, fetchUpcoming } from "@/lib/landing";
import "./landing.css";

/** Content changes go live within a minute of an administrator publishing. */
export const revalidate = 60;

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: `campVote · Secure ${CAMPUS_NAME} elections`,
  description: `Guild and campus elections at ${CAMPUS_NAME}, end to end: eligibility, secret ballots, live turnout and certified results, on any phone.`,
  openGraph: {
    type: "website",
    siteName: "campVote",
    title: `campVote · Secure ${CAMPUS_NAME} elections`,
    description:
      "Vote once. Keep it secret. Trust the result. Campus elections end to end, on any phone.",
    images: [
      {
        url: "/og.png",
        width: 1200,
        height: 630,
        alt: "campVote — secure campus elections",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: `campVote · Secure ${CAMPUS_NAME} elections`,
    description: "Vote once. Keep it secret. Trust the result.",
    images: ["/og.png"],
  },
  icons: {
    icon: "/icon.svg",
    apple: "/apple-icon.svg",
  },
};

export default async function Home() {
  const [document, upcoming] = await Promise.all([fetchLanding(), fetchUpcoming()]);
  return (
    <>
      <LandingSignedInRedirect />
      <LandingPage
        document={document}
        upcoming={upcoming.elections}
        upcomingFailed={upcoming.failed}
      />
    </>
  );
}
