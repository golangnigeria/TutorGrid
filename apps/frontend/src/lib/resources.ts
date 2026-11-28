import { API_URL } from "@/config/env";
import { AuthApiError } from "@/lib/auth";

export const MAX_RESOURCE_SIZE_BYTES = 25 * 1024 * 1024; // 25 MiB

export const ALLOWED_RESOURCE_MIME_TYPES = [
  "image/png",
  "image/jpeg",
  "image/webp",
  "application/pdf",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
] as const;

export const ALLOWED_RESOURCE_ACCEPT =
  ".png,.jpg,.jpeg,.webp,.pdf,.ppt,.pptx,image/png,image/jpeg,image/webp,application/pdf,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation";

export type ResourceStatus = "pending" | "ready" | "failed" | "deleting";
export type MalwareScanStatus = "pending" | "skipped" | "clean" | "infected";
export type ResourceCategory = "all" | "documents" | "slides" | "images";

export interface ResourceItem {
  id: string;
  tenant_id: string;
  uploaded_by_user_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  etag?: string;
  status: ResourceStatus;
  malware_scan_status: MalwareScanStatus;
  created_at: string;
  updated_at: string;
}

export interface PresignedUploadResponse {
  resource: ResourceItem;
  upload_url: string;
  expires_at: string;
}

export interface PresignedDownloadResponse {
  resource_id: string;
  filename: string;
  download_url: string;
  expires_at: string;
}

async function requestResourceJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
    ...init,
  });

  const payload = (await response.json().catch(() => ({}))) as T & {
    code?: string;
    message?: string;
    error?: { code?: string; message?: string };
  };

  if (!response.ok) {
    throw new AuthApiError(
      response.status,
      payload.error?.code ?? payload.code ?? "REQUEST_FAILED",
      payload.error?.message ?? payload.message ?? `Request failed (HTTP ${response.status})`,
    );
  }

  return payload;
}

export function validateFilePreFlight(file: File): string | null {
  if (file.size <= 0) {
    return "File cannot be empty.";
  }
  if (file.size > MAX_RESOURCE_SIZE_BYTES) {
    return "File exceeds the 25 MB MVP upload limit.";
  }
  const allowed = ALLOWED_RESOURCE_MIME_TYPES as readonly string[];
  if (!allowed.includes(file.type)) {
    return "Unsupported file type. Allowed formats: PNG, JPEG, WebP, PDF, PPT, PPTX.";
  }
  return null;
}

export function listResources(params?: {
  search?: string;
  category?: ResourceCategory;
  limit?: number;
  offset?: number;
}): Promise<{ resources: ResourceItem[] }> {
  const qs = new URLSearchParams();
  if (params?.search?.trim()) qs.set("search", params.search.trim());
  if (params?.category && params.category !== "all") qs.set("category", params.category);
  if (params?.limit) qs.set("limit", String(params.limit));
  if (params?.offset) qs.set("offset", String(params.offset));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return requestResourceJSON<{ resources: ResourceItem[] }>(`/api/v1/resources${suffix}`);
}

export function requestUploadURL(
  filename: string,
  contentType: string,
  sizeBytes: number,
): Promise<PresignedUploadResponse> {
  return requestResourceJSON<PresignedUploadResponse>("/api/v1/resources/upload-url", {
    method: "POST",
    body: JSON.stringify({
      filename,
      content_type: contentType,
      size_bytes: sizeBytes,
    }),
  });
}

export async function uploadFileDirectlyToB2(uploadUrl: string, file: File): Promise<void> {
  let response: Response;
  try {
    response = await fetch(uploadUrl, {
      method: "PUT",
      headers: {
        "Content-Type": file.type,
      },
      body: file,
    });
  } catch {
    throw new AuthApiError(
      0,
      "STORAGE_CORS_OR_NETWORK_ERROR",
      "Direct upload to Backblaze B2 was blocked by the browser (CORS or network error). Enable S3-compatible CORS rules for PUT on your Backblaze B2 bucket.",
    );
  }

  if (!response.ok) {
    throw new AuthApiError(
      response.status,
      "STORAGE_UPLOAD_FAILED",
      `Direct storage upload failed (HTTP ${response.status}). Check Backblaze B2 CORS and bucket permissions.`,
    );
  }
}

export function completeResourceUpload(resourceId: string): Promise<{ resource: ResourceItem }> {
  return requestResourceJSON<{ resource: ResourceItem }>(
    `/api/v1/resources/${encodeURIComponent(resourceId)}/complete`,
    { method: "POST" },
  );
}

export function getResourceDownloadURL(resourceId: string): Promise<PresignedDownloadResponse> {
  return requestResourceJSON<PresignedDownloadResponse>(
    `/api/v1/resources/${encodeURIComponent(resourceId)}/download-url`,
  );
}

export function deleteResource(resourceId: string): Promise<{ status: string }> {
  return requestResourceJSON<{ status: string }>(
    `/api/v1/resources/${encodeURIComponent(resourceId)}`,
    { method: "DELETE" },
  );
}
