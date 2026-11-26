"use client";

import { fileUrl } from "@/lib/api";

/** The format the in-app viewer should use, read from the stored file name. */
export function manifestoKind(url: string): "pdf" | "audio" | "video" | "file" {
  const path = url.split(/[?#]/)[0].toLowerCase();
  if (path.endsWith(".pdf")) return "pdf";
  if (path.endsWith(".mp3") || path.endsWith(".m4a")) return "audio";
  if (path.endsWith(".mp4")) return "video";
  return "file";
}

/**
 * Renders a candidate's manifesto in the page: a PDF frame, an audio player or
 * a video player. Only the three formats the upload accepts are expected, but
 * an older link falls back to a download instead of rendering nothing.
 */
export function ManifestoViewer({ url, title }: { url: string; title: string }) {
  const src = fileUrl(url);
  const kind = manifestoKind(url);

  if (kind === "pdf") {
    return (
      <iframe
        src={src}
        title={`${title} manifesto`}
        className="manifesto-viewer"
        style={{ width: "100%", height: 480, border: "1px solid var(--line)" }}
      />
    );
  }
  if (kind === "audio") {
    return (
      <audio controls src={src} style={{ width: "100%" }}>
        Your browser cannot play this audio manifesto.
      </audio>
    );
  }
  if (kind === "video") {
    return (
      <video
        controls
        src={src}
        style={{ width: "100%", maxHeight: 420, border: "1px solid var(--line)" }}
      >
        Your browser cannot play this video manifesto.
      </video>
    );
  }
  return (
    <a className="btn sm" href={src} target="_blank" rel="noreferrer">
      Open manifesto
    </a>
  );
}
