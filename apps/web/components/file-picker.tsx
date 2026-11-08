"use client";

import { useState, useRef } from "react";
import { Icon } from "./icon";
import { fileUrl } from "@/lib/api";

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
  const hasError = error || uploadError;

  return (
    <div className="field">
      <label htmlFor={`file-picker-${label.replace(/\s+/g, "-")}`}>{label}</label>
      {hint && <span className="hint">{hint}</span>}

      <div
        className={`affix file-picker ${hasError ? "bad" : ""} ${dragActive ? "drag-active" : ""}`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
      >
        <input
          ref={inputRef}
          type="file"
          id={`file-picker-${label.replace(/\s+/g, "-")}`}
          className="file-picker-input"
          accept={isImage ? "image/jpeg,image/png,image/gif,image/webp" : "application/pdf"}
          onChange={handleInputChange}
          disabled={disabled || uploading}
          aria-hidden="true"
          tabIndex={-1}
        />
        {!value ? (
          <div className="file-picker-empty">
            <div
              className={`file-picker-icon ${uploading ? "uploading" : ""} ${dragActive ? "drag" : ""}`}
            >
              {uploading ? (
                <Icon name="clock" size="l" />
              ) : (
                <Icon name={isImage ? "image" : "file"} size="l" />
              )}
            </div>
            <div className="file-picker-text">
              <span className="file-picker-label">
                {uploading
                  ? "Uploading..."
                  : dragActive
                    ? "Drop file here"
                    : `Select ${label.toLowerCase()}`}
              </span>
              <span className="hint">
                {isImage ? "JPEG, PNG, GIF, WebP" : "PDF"} · max {formatSize(maxSize)}
              </span>
            </div>
            <button
              type="button"
              className="btn primary sm"
              onClick={(e) => {
                e.stopPropagation();
                !disabled && !uploading && inputRef.current?.click();
              }}
              disabled={disabled || uploading}
            >
              Browse
            </button>
          </div>
        ) : (
          <div className="file-picker-filled">
            <div className="file-picker-preview-wrapper">
              {isImage ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={fileUrl(value)} alt="Preview" className="file-picker-preview" />
              ) : (
                <div className="file-picker-preview file-picker-preview-pdf">
                  <Icon name="file" size="s" />
                </div>
              )}
            </div>
            <div className="file-picker-filename">
              <span className="lbl">File uploaded</span>
              <span className="hint">{value.split("/").pop()}</span>
            </div>
            <div className="row" style={{ gap: 8 }}>
              <button
                type="button"
                className="btn sm ghost"
                onClick={(e) => {
                  e.stopPropagation();
                  handleReplace();
                }}
                disabled={disabled || uploading}
                title="Replace file"
              >
                <Icon name="refresh" size="s" />
                <span>Replace</span>
              </button>
              <button
                type="button"
                className="btn sm danger-o"
                onClick={(e) => {
                  e.stopPropagation();
                  handleRemove();
                }}
                disabled={disabled || uploading}
                title="Remove file"
              >
                <Icon name="x" size="s" />
              </button>
            </div>
          </div>
        )}
      </div>

      {hasError && <span className="err">{error || uploadError}</span>}
    </div>
  );
}
