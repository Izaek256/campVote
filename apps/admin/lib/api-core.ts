/**
 * Shared helpers for extracting human-readable error messages out of API
 * responses, axios errors, and plain Error objects.
 */
import type { AxiosError } from "axios";

interface ApiErrorBody {
  error?: {
    message?: string;
    code?: string;
  };
  message?: string;
}

/**
 * Returns the most useful message string from an unknown error value.
 *
 * Tries the API's standard `{ error: { message } }` envelope, then the
 * axios response body, then the Error .message, then falls back to the
 * provided `fallback`.
 */
export function getApiErrorMessage(err: unknown, fallback: string): string {
  if (!err) return fallback;

  // Axios error with a JSON response body.
  const axiosErr = err as AxiosError<ApiErrorBody>;
  if (axiosErr.response?.data) {
    const body = axiosErr.response.data;
    if (body.error?.message) return body.error.message;
    if (body.message) return body.message;
  }

  if (err instanceof Error) return err.message || fallback;

  return fallback;
}
