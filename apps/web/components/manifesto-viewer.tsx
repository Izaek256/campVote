"use client";

import { useRef, useState } from "react";
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
    return "audio/mpeg";
  }
  if (path.endsWith(".mov")) return "video/quicktime";
  return "video/mp4";
}

/**
 * Renders a candidate's manifesto inline: a PDF frame, an audio player or a
 * video player.
 *
 * - `key={src}` on every media element forces React to unmount/remount the DOM
 *   node when the URL changes so the browser issues a fresh request.
 * - `<source src type>` children let the browser decide playability before
 *   fetching, avoiding silent load failures.
 * - The PDF iframe carries `sandbox="allow-scripts allow-same-origin"` so the
 *   browser's built-in PDF viewer can run; without it Chrome blocks scripts
 *   inside the frame.
 * - The API serves files with `Accept-Ranges: bytes` and handles 206 partial
 *   content, so seeking inside large audio/video works.
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
        // allow-scripts lets the browser's PDF viewer run inside the frame.
        // allow-same-origin lets it read the blob the browser creates for the PDF.
        sandbox="allow-scripts allow-same-origin"
        style={{ width: "100%", height: 480, border: "1px solid var(--line)" }}
      />
    );
  }
  if (kind === "audio") {
    return (
      <audio key={src} controls style={{ width: "100%" }} preload="metadata">
        <source src={src} type={mimeFor("audio", url)} />
        Your browser cannot play this audio manifesto.{" "}
        <a href={src} download rel="noreferrer">
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
        <a href={src} download rel="noreferrer">
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

/**
 * A "Manifesto" button that behaves correctly for every file type:
 *
 * - PDF  → opens in a new tab (browser handles it natively, no CORS issue).
 * - Audio/video → opens an inline modal with the player so the file is fetched
 *   from the page's origin context, not from a detached new-tab null origin
 *   that the CORS policy blocks.
 *
 * Use this anywhere you previously had an <a href={fileUrl(url)} target="_blank">.
 */
export function ManifestoButton({
  url,
  title,
  className = "btn sm",
}: {
  url: string;
  title: string;
  className?: string;
}) {
  const src = fileUrl(url);
  const kind = manifestoKind(url);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [open, setOpen] = useState(false);

  if (kind === "pdf" || kind === "file") {
    // PDFs and unknown files open in a new tab; the browser handles them fine
    // and there is no CORS issue because they carry the right origin header.
    return (
      <a className={className} href={src} target="_blank" rel="noreferrer">
        Manifesto
      </a>
    );
  }

  // Audio or video — show inline in a modal to avoid the null-origin CORS block.
  function openModal() {
    setOpen(true);
    dialogRef.current?.showModal();
  }
  function closeModal() {
    setOpen(false);
    dialogRef.current?.close();
  }

  return (
    <>
      <button type="button" className={className} onClick={openModal}>
        Manifesto
      </button>
      <dialog
        ref={dialogRef}
        className="modal-dialog modal"
        style={{ maxWidth: 680 }}
        onClose={closeModal}
      >
        <div className="modal-h">
          <h3 style={{ margin: 0, fontSize: "1rem", fontWeight: 600 }}>{title}</h3>
          <button type="button" className="btn sm ghost" onClick={closeModal}>
            Close
          </button>
        </div>
        <div className="modal-b" style={{ padding: "12px 20px 20px" }}>
          {/* Only render the player while the modal is open so the browser
              releases the media resource as soon as the dialog closes. */}
          {open ? <ManifestoViewer url={url} title={title} /> : null}
        </div>
      </dialog>
    </>
  );
}
