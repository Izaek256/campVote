"use client";

import { fileUrl } from "@/lib/api";

/** The format the in-app viewer should use, read from the stored file name. */
export function manifestoKind(url: string): "pdf" | "audio" | "video" | "file" {
  const path = url.split(/[?#]/)[0].toLowerCase();
  if (path.endsWith(".pdf")) return "pdf";
  if (path.endsWith(".mp3") || path.endsWith(".m4a")) return "audio";
  if (path.endsWith(".mp4") || path.endsWith(".m4v") || path.endsWith(".mov")) return "video";
  return "file";
}

function mimeFor(kind: "audio" | "video", url: string): string {
  const path = url.split(/[?#]/)[0].toLowerCase();
  if (kind === "audio") {
    if (path.endsWith(".m4a")) return "audio/mp4";
    return "audio/mpeg"; // mp3
  }
  // video
  if (path.endsWith(".mov")) return "video/quicktime";
  return "video/mp4";
}

/**
 * Renders a candidate's manifesto in the page: a PDF frame, an audio player or
 * a video player. Only the three formats the upload accepts are expected, but
 * an older link falls back to a download instead of rendering nothing.
 *
 * Key notes:
 * - audio/video use a <source> child with an explicit MIME type so browsers
 *   do not have to sniff the content before deciding whether to load it.
 * - The `key` on each media element forces React to unmount and remount the
 *   DOM node when the URL changes, which makes the browser discard any cached
 *   (and potentially stale) media resource and start a fresh network request.
 * - Files are fetched directly from the API origin via fileUrl(), which
 *   rewrites the stored path to the correct host. The server must respond with
 *   Accept-Ranges: bytes so the browser can seek inside large audio/video files.
 */
export function ManifestoViewer({ url, title }: { url: string; title: string }) {
  const src = fileUrl(url);
  const kind = manifestoKind(url);

  if (kind === "pdf") {
    return (
      <iframe
        key={src}
        src={src}
        title={`${title} manifesto`}
        className="manifesto-viewer"
        style={{ width: "100%", height: 480, border: "1px solid var(--line)" }}
      />
    );
  }
  if (kind === "audio") {
    return (
      // key forces a DOM remount when the URL changes so the browser loads the
      // new resource rather than holding on to the previous media element state.
      <audio key={src} controls style={{ width: "100%" }} preload="metadata">
        <source src={src} type={mimeFor("audio", url)} />
        Your browser cannot play this audio manifesto.{" "}
        <a href={src} target="_blank" rel="noreferrer">
          Download instead
        </a>
        .
      </audio>
    );
  }
  if (kind === "video") {
    return (
      <video
        key={src}
        controls
        style={{ width: "100%", maxHeight: 420, border: "1px solid var(--line)" }}
        preload="metadata"
      >
        <source src={src} type={mimeFor("video", url)} />
        Your browser cannot play this video manifesto.{" "}
        <a href={src} target="_blank" rel="noreferrer">
          Download instead
        </a>
        .
      </video>
    );
  }
  return (
    <a className="btn sm" href={src} target="_blank" rel="noreferrer">
      Open manifesto
    </a>
  );
}
