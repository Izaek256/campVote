/**
 * Thin axios wrapper used by every admin page.
 *
 * All requests go to the backend at NEXT_PUBLIC_API_URL (defaults to
 * localhost:8080 for local development). Credentials (HttpOnly cookies) and
 * the CSRF double-submit token are included automatically.
 */
import axios from "axios";

const BASE_URL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

/** Read the CSRF cookie the server sets and return it for the request header. */
function csrfToken(): string {
  if (typeof document === "undefined") return "";
  const m = document.cookie.match(/(?:^|;\s*)grit_csrf=([^;]+)/);
  return m ? decodeURIComponent(m[1]) : "";
}

export const apiClient = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
});

// Attach the CSRF token to every mutating request.
apiClient.interceptors.request.use((config) => {
  const method = (config.method ?? "get").toLowerCase();
  if (method !== "get" && method !== "head") {
    const token = csrfToken();
    if (token) {
      config.headers = config.headers ?? {};
      config.headers["X-CSRF-Token"] = token;
    }
  }
  return config;
});

/**
 * Upload a file via the backend presign-then-PUT flow.
 *
 * Returns the axios response from the complete step so the caller can read
 * the resulting URL out of response.data.
 */
export async function uploadFile(file: File) {
  // Step 1: get a presigned URL and an object key.
  const { data: presignData } = await apiClient.post<{
    data: { presigned_url: string; key: string };
  }>("/api/v1/uploads/presign", {
    filename: file.name,
    content_type: file.type,
    size: file.size,
  });

  const { presigned_url, key } = presignData.data;

  // Step 2: PUT the file directly to the storage provider.
  await fetch(presigned_url, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });

  // Step 3: tell the backend the upload is done so it records the asset.
  return apiClient.post("/api/v1/uploads/complete", { key, filename: file.name });
}
