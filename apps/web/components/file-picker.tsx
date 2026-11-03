"use client";

import { useState, useRef } from "react";
import { Icon } from "./icon";

interface FilePickerProps {
  /** File type to accept (e.g., "image" for photos, "pdf" for documents) */
  accepts: "image" | "pdf";
  /** Maximum file size in bytes */
  maxSize: number;
  /** Current file URL (if already uploaded) */
  value?: string;
  /** Called when a file is selected and uploaded */
  onChange: (url: string) => void;
  /** Label shown above the picker */
  label: string;
  /** Helper text below the label */
  hint?: string;
  /** Disabled state */
  disabled?: boolean;
  /** Optional error message to display */
  error?: string;
}

export function FilePicker({
  accepts,
  maxSize,
  value,
  onChange,
  label,
  hint,
  disabled,
  error,
}: FilePickerProps) {
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [dragActive, setDragActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const validateFile = (file: File): string | null => {
    const isPdf = accepts === "pdf";
    const expectedType = isPdf ? "application/pdf" : "image/";

    if (!file.type.startsWith(expectedType)) {
      return isPdf
        ? "Please upload a PDF file"
        : "Please upload an image file (JPEG, PNG, GIF, WebP)";
    }

    if (file.size > maxSize) {
      return `File too large (max ${formatSize(maxSize)})`;
    }

    return null;
  };

  const handleFile = async (file: File) => {
    const validationError = validateFile(file);
    if (validationError) {
      setUploadError(validationError);
      return;
    }

    setUploadError("");
    setUploading(true);

    try {
      // Import upload function dynamically to avoid circular dependency
      const { upload } = await import("@/lib/api");
      const ref = await upload(file, { accepts, maxSize });
      onChange(ref.url);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Upload failed";
      setUploadError(message);
    } finally {
      setUploading(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (disabled || uploading) return;

    const file = e.dataTransfer.files?.[0];
    if (file) {
      handleFile(file);
    }
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFile(file);
    }
  };

  const handleRemove = () => {
    onChange("");
    setUploadError("");
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  const handleReplace = () => {
    inputRef.current?.click();
  };

  const isImage = accepts === "image";

  return (
    <div className="field">
      <label htmlFor={`file-picker-${label.replace(/\s+/g, "-")}`}>{label}</label>
      {hint && <span className="hint">{hint}</span>}

      {!value ? (
        <div
          className={`file-dropzone ${dragActive ? "active" : ""} ${error || uploadError ? "bad" : ""}`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
          onClick={() => !disabled && !uploading && inputRef.current?.click()}
          style={{
            border: "2px dashed var(--line)",
            borderRadius: 8,
            padding: 24,
            textAlign: "center",
            cursor: disabled || uploading ? "not-allowed" : "pointer",
            background: dragActive ? "var(--sunken)" : "",
          }}
        >
          <input
            ref={inputRef}
            type="file"
            accept={isImage ? "image/jpeg,image/png,image/gif,image/webp" : "application/pdf"}
            onChange={handleInputChange}
            disabled={disabled || uploading}
            style={{ display: "none" }}
          />
          <Icon name={isImage ? "image" : "file"} size="l" style={{ color: "var(--muted)" }} />
          <p style={{ marginTop: 12, marginBottom: 4 }}>
            {uploading ? "Uploading..." : dragActive ? "Drop file here" : "Drag & drop or click to browse"}
          </p>
          <span className="hint">
            {isImage ? "JPEG, PNG, GIF, WebP" : "PDF"} · max {formatSize(maxSize)}
          </span>
        </div>
      ) : (
        <div
          className="file-preview"
          style={{
            border: "1px solid var(--line)",
            borderRadius: 8,
            padding: 16,
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          {isImage ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={value}
              alt="Preview"
              style={{
                width: 64,
                height: 64,
                objectFit: "cover",
                borderRadius: 4,
                border: "1px solid var(--line)",
              }}
            />
          ) : (
            <div
              style={{
                width: 64,
                height: 64,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                background: "var(--sunken)",
                borderRadius: 4,
              }}
            >
              <Icon name="file" size="l" style={{ color: "var(--muted)" }} />
            </div>
          )}
          <div style={{ flex: 1, minWidth: 0 }}>
            <span className="hint" style={{ display: "block", marginBottom: 4 }}>
              Uploaded
            </span>
            <a
              href={value}
              target="_blank"
              rel="noreferrer"
              className="hint"
              style={{
                display: "block",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
                textDecoration: "underline",
              }}
            >
              {value.split("/").pop()}
            </a>
          </div>
          <div className="row" style={{ gap: 8 }}>
            <button
              type="button"
              className="btn sm"
              onClick={handleReplace}
              disabled={disabled || uploading}
            >
              <Icon name="refresh" size="s" />
              Replace
            </button>
            <button
              type="button"
              className="btn sm danger-o"
              onClick={handleRemove}
              disabled={disabled || uploading}
            >
              <Icon name="x" size="s" />
            </button>
          </div>
        </div>
      )}

      {(error || uploadError) && (
        <span className="err" style={{ display: "block", marginTop: 4 }}>
          {error || uploadError}
        </span>
      )}
    </div>
  );
}
