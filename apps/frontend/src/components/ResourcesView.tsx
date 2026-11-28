import { type ChangeEvent, type DragEvent, useCallback, useEffect, useRef, useState } from "react";
import {
  Download,
  FileImage,
  FileSpreadsheet,
  FileText,
  FolderOpen,
  Search,
  ShieldAlert,
  Trash2,
  UploadCloud,
} from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { AuthApiError, type MeResponse } from "@/lib/auth";
import {
  ALLOWED_RESOURCE_ACCEPT,
  completeResourceUpload,
  deleteResource,
  getResourceDownloadURL,
  listResources,
  requestUploadURL,
  uploadFileDirectlyToB2,
  validateFilePreFlight,
  type ResourceCategory,
  type ResourceItem,
} from "@/lib/resources";

interface ResourcesViewProps {
  me: MeResponse;
}

type UploadStage =
  | { state: "idle" }
  | { state: "preparing"; filename: string }
  | { state: "uploading"; filename: string }
  | { state: "verifying"; filename: string };

const categoryOptions: { id: ResourceCategory; label: string }[] = [
  { id: "all", label: "All" },
  { id: "documents", label: "PDFs" },
  { id: "slides", label: "Slides" },
  { id: "images", label: "Images" },
];

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function fileTypeLabel(mime: string): string {
  switch (mime) {
    case "application/pdf":
      return "PDF";
    case "application/vnd.ms-powerpoint":
      return "PPT";
    case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
      return "PPTX";
    case "image/png":
      return "PNG";
    case "image/jpeg":
      return "JPEG";
    case "image/webp":
      return "WebP";
    default:
      return "File";
  }
}

function ResourceIcon({ mime }: { mime: string }) {
  if (mime.startsWith("image/")) {
    return <FileImage className="h-4 w-4 text-[#8FAF94]" aria-hidden="true" />;
  }
  if (mime.includes("presentation") || mime.includes("powerpoint")) {
    return <FileSpreadsheet className="h-4 w-4 text-[#D6C29F]" aria-hidden="true" />;
  }
  return <FileText className="h-4 w-4 text-[#8FAF94]" aria-hidden="true" />;
}

export function ResourcesView({ me }: ResourcesViewProps) {
  const activeMembership =
    me.memberships.find((m) => m.tenant_id === me.active_tenant_id) ?? null;

  const isLearner = activeMembership?.role === "learner";
  const isAdmin =
    activeMembership?.role === "org_admin" ||
    activeMembership?.role === "platform_admin";

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [dragging, setDragging] = useState(false);
  const [uploadStage, setUploadStage] = useState<UploadStage>({ state: "idle" });
  const [uploadError, setUploadError] = useState<string | null>(null);

  const [resources, setResources] = useState<ResourceItem[]>([]);
  const [listState, setListState] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [listError, setListError] = useState<string | null>(null);

  const [search, setSearch] = useState("");
  const [category, setCategory] = useState<ResourceCategory>("all");
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadResources = useCallback(async () => {
    if (!activeMembership || isLearner) {
      setResources([]);
      setListState("idle");
      return;
    }
    setListState("loading");
    setListError(null);
    try {
      const res = await listResources({ search, category });
      setResources(res.resources);
      setListState("loaded");
    } catch (err) {
      setListState("error");
      setListError(
        err instanceof AuthApiError ? err.message : "Unable to load resource library.",
      );
    }
  }, [activeMembership, isLearner, search, category]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadResources();
    }, 150);
    return () => window.clearTimeout(timer);
  }, [loadResources]);

  const processFileUpload = async (file: File) => {
    setUploadError(null);
    const preflightError = validateFilePreFlight(file);
    if (preflightError) {
      setUploadError(preflightError);
      toast.error(preflightError);
      return;
    }

    try {
      setUploadStage({ state: "preparing", filename: file.name });
      const presigned = await requestUploadURL(file.name, file.type, file.size);

      setUploadStage({ state: "uploading", filename: file.name });
      await uploadFileDirectlyToB2(presigned.upload_url, file);

      setUploadStage({ state: "verifying", filename: file.name });
      await completeResourceUpload(presigned.resource.id);

      toast.success(`Uploaded ${file.name}`, {
        description: "Verified and stored privately in Backblaze B2.",
      });
      await loadResources();
    } catch (err) {
      const message =
        err instanceof AuthApiError ? err.message : "Upload failed. Please try again.";
      setUploadError(message);
      toast.error(message);
    } finally {
      setUploadStage({ state: "idle" });
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      void processFileUpload(file);
    }
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setDragging(false);
    if (uploadStage.state !== "idle") return;
    const file = e.dataTransfer.files?.[0];
    if (file) {
      void processFileUpload(file);
    }
  };

  // Strictly download via Content-Disposition: attachment — NEVER render unscanned files inline.
  const handleDownload = async (item: ResourceItem) => {
    setDownloadingId(item.id);
    try {
      const res = await getResourceDownloadURL(item.id);
      const link = document.createElement("a");
      link.href = res.download_url;
      link.download = res.filename;
      link.rel = "noopener noreferrer";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      toast.error(err instanceof AuthApiError ? err.message : "Unable to prepare download link.");
    } finally {
      setDownloadingId(null);
    }
  };

  const handleDelete = async (item: ResourceItem) => {
    setDeletingId(item.id);
    try {
      await deleteResource(item.id);
      setResources((prev) => prev.filter((r) => r.id !== item.id));
      toast.success(`Deleted ${item.filename}`);
    } catch (err) {
      toast.error(err instanceof AuthApiError ? err.message : "Unable to delete resource.");
    } finally {
      setDeletingId(null);
    }
  };

  if (!activeMembership) {
    return (
      <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
        <h1 className="font-display text-2xl font-semibold text-[#F3F0E8]">Resource Library</h1>
        <p className="mt-2 text-sm text-[#A6B0A8]">
          Select an active Teaching Space in the top-left workspace pill to manage resources.
        </p>
      </section>
    );
  }

  if (isLearner) {
    return (
      <section className="mx-auto max-w-xl rounded-2xl border border-[#D6C29F]/30 bg-[#181F1A]/84 p-6 backdrop-blur-md">
        <div className="flex items-center gap-2.5 text-[#D6C29F]">
          <ShieldAlert className="h-5 w-5" aria-hidden="true" />
          <h1 className="font-display text-xl font-semibold text-[#F3F0E8]">
            Teacher &amp; Admin Library Only
          </h1>
        </div>
        <p className="mt-2 text-sm leading-relaxed text-[#A6B0A8]">
          The workspace Resource Library is restricted to teachers and organization admins.
          Reference materials attached to published lessons will be available inside your lessons
          and live classrooms.
        </p>
      </section>
    );
  }

  const isUploading = uploadStage.state !== "idle";

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 28 }}
      className="space-y-6"
    >
      {/* Header & Upload Dropzone */}
      <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="font-display text-2xl font-semibold tracking-tight text-[#F3F0E8]">
              Resource Library
            </h1>
            <p className="mt-1 text-sm text-[#A6B0A8]">
              Upload reference PDFs, slides, and images directly to private Backblaze B2 storage for{" "}
              <span className="text-[#F3F0E8]">{activeMembership.tenant_name}</span>.
            </p>
          </div>
          <span className="inline-flex w-fit items-center rounded-full border border-[#D6C29F]/30 bg-[#D6C29F]/10 px-3 py-1 text-[11px] font-medium text-[#D6C29F]">
            Downloads only · Malware scan pending
          </span>
        </div>

        <div
          onDragOver={(e) => {
            e.preventDefault();
            if (!isUploading) setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={handleDrop}
          className={`mt-5 flex flex-col items-center justify-center rounded-2xl border border-dashed px-6 py-8 text-center transition-colors ${
            dragging
              ? "border-[#8FAF94] bg-[#8FAF94]/12"
              : "border-[#F3F0E8]/20 bg-[#101411]/60 hover:border-[#8FAF94]/50"
          }`}
        >
          <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-[#8FAF94]/35 bg-[#8FAF94]/12 text-[#8FAF94]">
            <UploadCloud className="h-5 w-5" aria-hidden="true" />
          </div>

          {uploadStage.state === "idle" ? (
            <>
              <p className="mt-3 text-sm font-medium text-[#F3F0E8]">
                Drag and drop a file here, or browse from your device
              </p>
              <p className="mt-1 text-xs text-[#A6B0A8]">
                PNG, JPEG, WebP, PDF, PPT, or PPTX · Up to 25 MB · Stored as reference material
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept={ALLOWED_RESOURCE_ACCEPT}
                onChange={handleFileChange}
                className="sr-only"
                id="resource-file-input"
              />
              <motion.button
                whileTap={{ scale: 0.985 }}
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="mt-4 inline-flex min-h-10 cursor-pointer items-center justify-center rounded-xl bg-[#8FAF94] px-5 py-2 text-xs font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
              >
                Choose File to Upload
              </motion.button>
            </>
          ) : (
            <div role="status" className="mt-3 flex flex-col items-center gap-1.5">
              <span className="h-2 w-2 animate-pulse rounded-full bg-[#8FAF94]" />
              <p className="text-sm font-medium text-[#F3F0E8]">
                {uploadStage.state === "preparing" && `Preparing secure upload for ${uploadStage.filename}...`}
                {uploadStage.state === "uploading" && `Uploading ${uploadStage.filename} directly to Backblaze B2...`}
                {uploadStage.state === "verifying" && `Verifying file signature & package structure...`}
              </p>
            </div>
          )}
        </div>

        {uploadError && (
          <p
            role="alert"
            className="mt-4 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-4 py-2.5 text-sm text-[#F4C2C2]"
          >
            {uploadError}
          </p>
        )}
      </section>

      {/* Search, Filter & Resource List */}
      <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <FolderOpen className="h-4 w-4 text-[#8FAF94]" aria-hidden="true" />
            <h2 className="text-base font-semibold text-[#F3F0E8]">Uploaded Resources</h2>
          </div>

          <div className="flex flex-col gap-2.5 sm:flex-row sm:items-center">
            <div className="relative">
              <Search
                className="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-[#A6B0A8]"
                aria-hidden="true"
              />
              <input
                type="search"
                aria-label="Search resources"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by filename..."
                className="min-h-10 w-full rounded-xl border border-[#F3F0E8]/16 bg-[#101411]/85 py-1.5 pr-3 pl-9 text-xs text-[#F3F0E8] placeholder:text-[#7E8A80] focus-visible:border-[#8FAF94] focus-visible:outline-2 focus-visible:outline-[#8FAF94]/40 sm:w-52"
              />
            </div>

            <div className="flex flex-wrap items-center gap-1 rounded-xl border border-[#F3F0E8]/12 bg-[#101411]/70 p-1">
              {categoryOptions.map((opt) => {
                const active = category === opt.id;
                return (
                  <button
                    key={opt.id}
                    type="button"
                    onClick={() => setCategory(opt.id)}
                    className={`cursor-pointer rounded-lg px-2.5 py-1 text-xs font-medium transition-colors ${
                      active
                        ? "bg-[#8FAF94]/20 text-[#F3F0E8]"
                        : "text-[#A6B0A8] hover:text-[#F3F0E8]"
                    }`}
                  >
                    {opt.label}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {listState === "loading" && (
          <p role="status" className="mt-5 text-sm text-[#A6B0A8]">
            Loading resources...
          </p>
        )}

        {listState === "error" && (
          <p
            role="alert"
            className="mt-5 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-4 py-3 text-sm text-[#F4C2C2]"
          >
            {listError}
          </p>
        )}

        {listState === "loaded" && resources.length === 0 && (
          <div className="mt-5 rounded-xl border border-[#F3F0E8]/10 bg-[#101411]/55 px-4 py-8 text-center">
            <p className="text-sm font-medium text-[#F3F0E8]">No resources found</p>
            <p className="mt-1 text-xs text-[#A6B0A8]">
              {search || category !== "all"
                ? "Try clearing your search or category filter."
                : "Upload your first PDF, PowerPoint deck, or image above."}
            </p>
          </div>
        )}

        {listState === "loaded" && resources.length > 0 && (
          <ul className="mt-4 divide-y divide-[#F3F0E8]/10">
            {resources.map((item) => {
              const canDelete = isAdmin || item.uploaded_by_user_id === me.user.id;
              return (
                <li
                  key={item.id}
                  className="flex flex-col gap-3 py-3.5 text-sm sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-[#F3F0E8]/12 bg-[#101411]/80">
                      <ResourceIcon mime={item.content_type} />
                    </div>
                    <div className="min-w-0">
                      <p className="truncate font-medium text-[#F3F0E8]">{item.filename}</p>
                      <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[#A6B0A8]">
                        <span className="rounded-full border border-[#F3F0E8]/14 bg-[#222B24] px-2 py-0.5 text-[11px] font-medium text-[#E3E1D8]">
                          {fileTypeLabel(item.content_type)}
                        </span>
                        <span>{formatBytes(item.size_bytes)}</span>
                        <span>·</span>
                        <span>{formatDate(item.created_at)}</span>
                        {item.malware_scan_status === "pending" && (
                          <span className="rounded-full border border-[#D6C29F]/30 bg-[#D6C29F]/10 px-2 py-0.5 text-[10px] font-medium text-[#D6C29F]">
                            Unscanned
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex shrink-0 items-center gap-2 self-start sm:self-center">
                    <button
                      type="button"
                      disabled={downloadingId === item.id}
                      onClick={() => void handleDownload(item)}
                      className="inline-flex min-h-9 cursor-pointer items-center gap-1.5 rounded-lg border border-[#8FAF94]/35 bg-[#8FAF94]/12 px-3 py-1.5 text-xs font-medium text-[#8FAF94] transition-colors hover:bg-[#8FAF94]/22 disabled:opacity-50"
                    >
                      <Download className="h-3.5 w-3.5" aria-hidden="true" />
                      {downloadingId === item.id ? "Preparing..." : "Download"}
                    </button>

                    {canDelete && (
                      <button
                        type="button"
                        disabled={deletingId === item.id}
                        onClick={() => void handleDelete(item)}
                        className="inline-flex min-h-9 cursor-pointer items-center gap-1.5 rounded-lg border border-[#D97373]/30 bg-[#D97373]/10 px-3 py-1.5 text-xs font-medium text-[#F4C2C2] transition-colors hover:bg-[#D97373]/20 disabled:opacity-50"
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                        {deletingId === item.id ? "Deleting..." : "Delete"}
                      </button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </motion.div>
  );
}
