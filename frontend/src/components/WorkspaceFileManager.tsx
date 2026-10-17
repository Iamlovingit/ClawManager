import { CompactTimestamp } from "./CompactTimestamp";
import { WorkspaceTextEditor, editableWorkspaceName, type WorkspaceText } from "./WorkspaceTextEditor";
import "../styles/workspace-time.css";
import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Download,
  Eye,
  File,
  FileText,
  Folder,
  FolderPlus,
  FolderUp,
  FilePenLine,
  TextCursorInput,
  RefreshCw,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { useI18n } from "../contexts/I18nContext";
import { interpolate, translate, type Locale } from "../lib/i18n";
import { workspaceService } from "../services/workspaceService";
import type { WorkspaceEntry, WorkspacePreview } from "../types/workspace";

interface WorkspaceFileManagerProps {
  instanceId: number;
  initialPath?: string;
  onMutation?: () => void | Promise<void>;
  refreshKey?: number;
  service?: WorkspaceFileOperations;
  workspaceKey?: string | number;
  canWrite?: boolean;
  localeOverride?: Locale;
  onSelectDirectory?: (path: string) => void | Promise<void>;
  selectingDirectory?: boolean;
}

export interface WorkspaceFileOperations {
  readText?(instanceId: number, path: string, signal?: AbortSignal): Promise<WorkspaceText>;
  saveText?(instanceId: number, path: string, text: string, version: string): Promise<WorkspaceText>;
  list(instanceId: number, path?: string): Promise<WorkspaceEntry[]>;
  preview(instanceId: number, path: string): Promise<WorkspacePreview>;
  previewBlob(instanceId: number, path: string): Promise<Blob>;
  downloadBlob(instanceId: number, path: string): Promise<Blob>;
  upload(instanceId: number, path: string, file: File): Promise<void>;
  mkdir(instanceId: number, path: string): Promise<void>;
  rename(instanceId: number, oldPath: string, newPath: string): Promise<void>;
  remove(instanceId: number, path: string): Promise<void>;
}

const invalidNamePattern = /[\\/]/;
const workspacePageSize = 250;

function joinPath(base: string, name: string) {
  return [base, name].filter(Boolean).join("/");
}

function parentPath(path: string) {
  const parts = path.split("/").filter(Boolean);
  parts.pop();
  return parts.join("/");
}

function fileName(path: string) {
  const parts = path.split("/").filter(Boolean);
  return parts.at(-1) ?? "";
}

function normalizeWorkspacePath(path: string | undefined) {
  const raw = path?.trim() ?? "";
  if (!raw || raw === "." || raw === "/") {
    return "";
  }
  if (raw.startsWith("/")) {
    return "";
  }
  return raw
    .replaceAll("\\", "/")
    .split("/")
    .filter(Boolean)
    .join("/");
}

function virtualRootBreadcrumbs(initialPath: string | undefined) {
  const raw = initialPath?.trim() ?? "";
  if (!raw.startsWith("/")) {
    return [];
  }
  return raw.split("/").filter(Boolean);
}

function breadcrumbItems(
  path: string,
  initialPath: string | undefined,
  workspaceLabel: string,
) {
  const parts = path.split("/").filter(Boolean);
  return [
    { key: "workspace", label: workspaceLabel, path: "" },
    ...virtualRootBreadcrumbs(initialPath).map((part, index) => ({
      key: `virtual-root-${index}-${part}`,
      label: part,
      path: "",
    })),
    ...parts.map((part, index) => ({
      key: `path-${index}-${part}`,
      label: part,
      path: parts.slice(0, index + 1).join("/"),
    })),
  ];
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function WorkspaceEntryName({ name }: { name: string }) {
  const nameRef = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  useEffect(() => {
    const element = nameRef.current;
    if (!element) {
      return;
    }

    const updateTruncation = () => {
      setIsTruncated(element.scrollWidth > element.clientWidth);
    };

    updateTruncation();
    const observer = new ResizeObserver(updateTruncation);
    observer.observe(element);
    return () => observer.disconnect();
  }, [name]);

  return (
    <span className="group relative min-w-0">
      <span ref={nameRef} className="block truncate font-medium text-slate-900">
        {name}
      </span>
      {isTruncated && (
        <span
          role="tooltip"
          className="pointer-events-none absolute left-0 top-full z-30 mt-1 w-max max-w-[24rem] whitespace-normal rounded-md bg-slate-800 px-2 py-1 text-xs font-normal text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
        >
          {name}
        </span>
      )}
    </span>
  );
}

function downloadBlob(blob: Blob, name: string) {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name || "download";
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

function assertEntryName(name: string) {
  const trimmed = name.trim();
  if (!trimmed || trimmed === "." || trimmed === ".." || invalidNamePattern.test(trimmed)) {
    return null;
  }
  return trimmed;
}

function normalizeUploadRelativePath(path: string) {
  const parts = path
    .replaceAll("\\", "/")
    .split("/")
    .map((part) => part.trim())
    .filter(Boolean);
  if (
    parts.length === 0 ||
    parts.some((part) => part === "." || part === ".." || invalidNamePattern.test(part))
  ) {
    return null;
  }
  return parts.join("/");
}

function collectFolderUploadDirectories(files: File[]) {
  const directories = new Set<string>();
  files.forEach((file) => {
    const relativePath = normalizeUploadRelativePath(file.webkitRelativePath || file.name);
    if (!relativePath) {
      return;
    }
    const directory = parentPath(relativePath);
    if (!directory) {
      return;
    }
    const parts = directory.split("/").filter(Boolean);
    parts.forEach((_, index) => {
      directories.add(parts.slice(0, index + 1).join("/"));
    });
  });
  return Array.from(directories).sort((left, right) => {
    const leftDepth = left.split("/").length;
    const rightDepth = right.split("/").length;
    return leftDepth === rightDepth ? left.localeCompare(right) : leftDepth - rightDepth;
  });
}

function getErrorMessage(err: unknown, fallback: string) {
  const responseError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
  if (responseError) {
    return responseError;
  }
  return err instanceof Error ? err.message : fallback;
}

function isWorkspaceEntryExistsError(err: unknown) {
  const message = getErrorMessage(err, "").toLowerCase();
  return message.includes("already exists") || message.includes("entry already exists");
}

function EntryIcon({ entry }: { entry: WorkspaceEntry }) {
  if (entry.is_dir) {
    return <Folder className="h-4 w-4 text-slate-500" />;
  }
  if (entry.previewable) {
    return <FileText className="h-4 w-4 text-slate-500" />;
  }
  return <File className="h-4 w-4 text-slate-500" />;
}

export function WorkspaceFileManager({
  instanceId,
  initialPath,
  onMutation,
  refreshKey,
  service = workspaceService,
  workspaceKey,
  canWrite = true,
  localeOverride,
  onSelectDirectory,
  selectingDirectory = false,
}: WorkspaceFileManagerProps) {
  const { locale, t } = useI18n();
  const effectiveLocale = localeOverride ?? locale;
  const translateLabel = useCallback(
    (key: string, variables?: Record<string, string | number>) => {
      const overridden = localeOverride ? translate(localeOverride, key) : undefined;
      return overridden ? interpolate(overridden, variables) : t(key, variables);
    },
    [localeOverride, t],
  );
  const queryClient = useQueryClient();
  const cacheKey = workspaceKey ?? instanceId;
  const uploadInputRef = useRef<HTMLInputElement | null>(null);
  const folderUploadInputRef = useRef<HTMLInputElement | null>(null);
  const uploadMenuRef = useRef<HTMLDivElement | null>(null);
  const [currentPath, setCurrentPath] = useState(() => normalizeWorkspacePath(initialPath));
  const [previewPath, setPreviewPath] = useState<string | null>(null);
  const [editPath, setEditPath] = useState<string | null>(null);
  const canEditEntry = (entry: WorkspaceEntry) => Boolean(service.readText && service.saveText && !entry.is_dir && entry.size <= 1024*1024 && editableWorkspaceName(entry.name));
  const previewObjectUrlRef = useRef<string | null>(null);
  const [previewObjectUrl, setPreviewObjectUrl] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [uploadStatus, setUploadStatus] = useState<string | null>(null);
  const [uploadMenuOpen, setUploadMenuOpen] = useState(false);
  const [pageIndex, setPageIndex] = useState(0);
  useEffect(() => {
    setCurrentPath(normalizeWorkspacePath(initialPath));
    setPreviewPath(null);
	setEditPath(null);
	setPageIndex(0);
  }, [cacheKey, initialPath]);

  useEffect(() => {
    if (!uploadMenuOpen) {
      return;
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (uploadMenuRef.current?.contains(event.target as Node)) {
        return;
      }
      setUploadMenuOpen(false);
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setUploadMenuOpen(false);
      }
    };
    window.addEventListener("pointerdown", handlePointerDown);
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("pointerdown", handlePointerDown);
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [uploadMenuOpen]);

  const entriesQuery = useQuery({
    queryKey: ["workspace", cacheKey, currentPath],
    queryFn: () => service.list(instanceId, currentPath),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    refetchOnWindowFocus: Boolean(service.readText),
    refetchInterval: service.readText ? 5000 : false,
  });

  const previewQuery = useQuery({
    queryKey: ["workspace-preview", cacheKey, previewPath],
    queryFn: () => service.preview(instanceId, previewPath ?? ""),
    enabled: Boolean(previewPath),
    staleTime: 15_000,
    gcTime: 5 * 60_000,
    refetchOnWindowFocus: Boolean(service.readText),
    refetchInterval: service.readText ? 5000 : false,
  });

  useEffect(() => {
    let disposed = false;
    if (previewObjectUrlRef.current) {
      window.URL.revokeObjectURL(previewObjectUrlRef.current);
      previewObjectUrlRef.current = null;
      setPreviewObjectUrl(null);
    }

    const preview = previewQuery.data;
    if (!previewPath || !preview || (preview.kind !== "image" && preview.kind !== "pdf")) {
      return;
    }

    service
      .previewBlob(instanceId, previewPath)
      .then((blob) => {
        if (!disposed) {
          const url = window.URL.createObjectURL(blob);
          previewObjectUrlRef.current = url;
          setPreviewObjectUrl(url);
        }
      })
      .catch((err: unknown) => {
        if (!disposed) {
          setError(getErrorMessage(err, translateLabel("workspaceFileManager.failedPreview")));
        }
      });

    return () => {
      disposed = true;
    };
  }, [instanceId, previewPath, previewQuery.data, service, translateLabel]);

  useEffect(() => {
    return () => {
      if (previewObjectUrlRef.current) {
        window.URL.revokeObjectURL(previewObjectUrlRef.current);
        previewObjectUrlRef.current = null;
      }
    };
  }, []);

  const invalidateCurrentPath = async () => {
    await queryClient.invalidateQueries({
      queryKey: ["workspace", cacheKey, currentPath],
    });
  };

  useEffect(() => {
    if (refreshKey === undefined) {
      return;
    }
    void queryClient.invalidateQueries({
      queryKey: ["workspace", cacheKey, currentPath],
    });
  }, [cacheKey, currentPath, queryClient, refreshKey]);

  const notifyMutation = async () => {
    await onMutation?.();
  };

  const runAction = async (key: string, action: () => Promise<void>) => {
    try {
      setBusyAction(key);
      setError(null);
      await action();
    } catch (err: unknown) {
      setError(getErrorMessage(err, translateLabel("workspaceFileManager.actionFailed")));
    } finally {
      setBusyAction(null);
      setUploadStatus(null);
    }
  };

  const clearUploadInputs = () => {
    if (uploadInputRef.current) {
      uploadInputRef.current.value = "";
    }
    if (folderUploadInputRef.current) {
      folderUploadInputRef.current.value = "";
    }
  };

  const handleUploadFiles = (fileList?: FileList | null) => {
    const files = Array.from(fileList ?? []);
    if (files.length === 0) {
      return;
    }
    void runAction("upload", async () => {
      for (const [index, file] of files.entries()) {
        setUploadStatus(
          translateLabel("workspaceFileManager.uploading", {
            current: index + 1,
            total: files.length,
          }),
        );
        await service.upload(instanceId, currentPath, file);
      }
      await invalidateCurrentPath();
      await notifyMutation();
      clearUploadInputs();
    });
  };

  const handleUploadFolder = (fileList?: FileList | null) => {
    const files = Array.from(fileList ?? []);
    if (files.length === 0) {
      return;
    }
    void runAction("upload-folder", async () => {
      const directories = collectFolderUploadDirectories(files);
      for (const [index, directory] of directories.entries()) {
        setUploadStatus(
          translateLabel("workspaceFileManager.creatingFolders", {
            current: index + 1,
            total: directories.length,
          }),
        );
        try {
          await service.mkdir(instanceId, joinPath(currentPath, directory));
        } catch (err: unknown) {
          if (!isWorkspaceEntryExistsError(err)) {
            throw err;
          }
        }
      }

      for (const [index, file] of files.entries()) {
        const relativePath = normalizeUploadRelativePath(file.webkitRelativePath || file.name);
        if (!relativePath) {
          continue;
        }
        setUploadStatus(
          translateLabel("workspaceFileManager.uploading", {
            current: index + 1,
            total: files.length,
          }),
        );
        await service.upload(
          instanceId,
          joinPath(currentPath, parentPath(relativePath)),
          file,
        );
      }
      await invalidateCurrentPath();
      await notifyMutation();
      clearUploadInputs();
    });
  };

  const openFileUploadPicker = () => {
    setUploadMenuOpen(false);
    uploadInputRef.current?.click();
  };

  const openFolderUploadPicker = () => {
    setUploadMenuOpen(false);
    folderUploadInputRef.current?.click();
  };

  const handleMkdir = () => {
    const name = assertEntryName(
      window.prompt(translateLabel("workspaceFileManager.folderNamePrompt")) || "",
    );
    if (!name) {
      return;
    }
    void runAction("mkdir", async () => {
      await service.mkdir(instanceId, joinPath(currentPath, name));
      await invalidateCurrentPath();
      await notifyMutation();
    });
  };

  const handleRename = (entry: WorkspaceEntry) => {
    const name = assertEntryName(
      window.prompt(translateLabel("workspaceFileManager.renamePrompt"), entry.name) || "",
    );
    if (!name || name === entry.name) {
      return;
    }
    void runAction(`rename:${entry.path}`, async () => {
      await service.rename(instanceId, entry.path, joinPath(parentPath(entry.path), name));
      if (previewPath === entry.path) {
        setPreviewPath(null);
      }
      await invalidateCurrentPath();
      await notifyMutation();
    });
  };

  const handleDelete = (entry: WorkspaceEntry) => {
    if (
      !window.confirm(
        translateLabel("workspaceFileManager.deleteConfirm", { name: entry.name }),
      )
    ) {
      return;
    }
    void runAction(`delete:${entry.path}`, async () => {
      await service.remove(instanceId, entry.path);
      if (previewPath === entry.path || previewPath?.startsWith(`${entry.path}/`)) {
        setPreviewPath(null);
      }
      await invalidateCurrentPath();
      await notifyMutation();
    });
  };

  const handleDownload = (entry: WorkspaceEntry) => {
    void runAction(`download:${entry.path}`, async () => {
      const blob = await service.downloadBlob(instanceId, entry.path);
      const name = entry.name || fileName(entry.path);
      downloadBlob(blob, entry.is_dir ? `${name}.zip` : name);
    });
  };

  const preview = previewQuery.data;
  const entries = entriesQuery.data ?? [];
  const pageCount = Math.max(1, Math.ceil(entries.length / workspacePageSize));
  const visiblePageIndex = Math.min(pageIndex, pageCount - 1);
  const visibleEntries = entries.slice(
    visiblePageIndex * workspacePageSize,
    (visiblePageIndex + 1) * workspacePageSize,
  );

  return (
    <section className="cm-surface flex h-full min-h-0 min-w-0 flex-col overflow-hidden max-xl:min-h-[360px]">
      <div className="flex items-center justify-between gap-3 border-b border-slate-200 px-3 py-2">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-1 text-sm">
            {breadcrumbItems(
              currentPath,
              initialPath,
              translateLabel("workspaceFileManager.workspace"),
            ).map((item, index, items) => (
              <span key={item.key} className="flex min-w-0 items-center gap-1">
                <button
                  type="button"
                  className="max-w-[140px] truncate rounded px-1.5 py-1 font-medium text-slate-700 hover:bg-slate-100"
                  onClick={() => {
                    setCurrentPath(item.path);
                    setPageIndex(0);
                  }}
                >
                  {item.label}
                </button>
                {index < items.length - 1 && <span className="text-slate-400">/</span>}
              </span>
            ))}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {canWrite && (
            <>
              <input
                ref={uploadInputRef}
                type="file"
                className="hidden"
                multiple
                onChange={(event) => handleUploadFiles(event.target.files)}
              />
              <input
                ref={folderUploadInputRef}
                type="file"
                className="hidden"
                multiple
                {...{ webkitdirectory: "", directory: "" }}
                onChange={(event) => handleUploadFolder(event.target.files)}
              />
            </>
          )}
          <button
            type="button"
            className="cm-icon-button"
            title={translateLabel("workspaceFileManager.refresh")}
            onClick={() => void invalidateCurrentPath()}
          >
            <RefreshCw className={`h-4 w-4 ${entriesQuery.isFetching ? "animate-spin" : ""}`} />
          </button>
          {onSelectDirectory && (
            <button
              type="button"
              className="rounded-md border border-blue-200 bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-60"
              title={translateLabel("workspaceFileManager.setAsProjectTitle")}
              disabled={selectingDirectory}
              onClick={() => void onSelectDirectory(currentPath)}
            >
              {selectingDirectory
                ? translateLabel("workspaceFileManager.settingProject")
                : translateLabel("workspaceFileManager.setAsProject")}
            </button>
          )}
          {canWrite && (
            <>
              <button
                type="button"
                className="cm-icon-button"
                title={translateLabel("workspaceFileManager.newFolder")}
                onClick={handleMkdir}
              >
                <FolderPlus className="h-4 w-4" />
              </button>
              <div ref={uploadMenuRef} className="relative">
                <button
                  type="button"
                  className="cm-icon-button"
                  title={translateLabel("workspaceFileManager.upload")}
                  aria-haspopup="menu"
                  aria-expanded={uploadMenuOpen}
                  disabled={busyAction === "upload" || busyAction === "upload-folder"}
                  onClick={() => setUploadMenuOpen((open) => !open)}
                >
                  <Upload className="h-4 w-4" />
                </button>
                {uploadMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 top-full z-20 mt-2 w-40 rounded-md border border-slate-200 bg-white py-1 shadow-lg"
                  >
                    <button
                      type="button"
                      role="menuitem"
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-50"
                      onClick={openFileUploadPicker}
                    >
                      <Upload className="h-4 w-4 text-slate-500" />
                      <span>{translateLabel("workspaceFileManager.uploadFiles")}</span>
                    </button>
                    <button
                      type="button"
                      role="menuitem"
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-50"
                      onClick={openFolderUploadPicker}
                    >
                      <FolderUp className="h-4 w-4 text-slate-500" />
                      <span>{translateLabel("workspaceFileManager.uploadFolder")}</span>
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {uploadStatus && (
        <div className="border-b border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-700">
          {uploadStatus}
        </div>
      )}

      {(error || entriesQuery.isError) && (
        <div className="border-b border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error || getErrorMessage(entriesQuery.error, translateLabel("workspaceFileManager.failedList"))}
          {entriesQuery.isError && <button type="button" disabled={entriesQuery.isFetching} onClick={() => void entriesQuery.refetch()} className="ml-3 rounded border px-2 py-1 disabled:opacity-50">{translateLabel("common.retry")}</button>}
        </div>
      )}

      <div className="flex min-h-0 flex-1 flex-col">
        <div className={`workspace-files-container ${previewPath ? "hidden" : "min-h-0 flex-1 overflow-y-auto overflow-x-hidden"}`}>
          {entriesQuery.isLoading ? (
            <div className="flex h-48 items-center justify-center text-sm text-slate-500">
              {translateLabel("workspaceFileManager.loading")}
            </div>
          ) : entriesQuery.isError && !entriesQuery.data ? (
            <div className="flex h-48 items-center justify-center text-sm text-slate-500">
              <button type="button" disabled={entriesQuery.isFetching} onClick={() => void entriesQuery.refetch()} className="rounded border px-3 py-2 disabled:opacity-50">
                {translateLabel("common.retry")}
              </button>
            </div>
          ) : entries.length === 0 ? (
            <div className="flex h-48 items-center justify-center text-sm text-slate-500">
              {translateLabel("workspaceFileManager.noFiles")}
            </div>
          ) : (
            <>
            <table className="workspace-files-table w-full table-fixed divide-y divide-slate-200 text-sm">
              <thead className="bg-slate-50 text-left text-xs font-medium uppercase tracking-normal text-slate-500">
                <tr>
                  <th className="px-3 py-2">{translateLabel("workspaceFileManager.name")}</th>
                  <th className="w-14 px-2 py-2">
                    {translateLabel("workspaceFileManager.size")}
                  </th>
                  <th className="workspace-time-column w-28 px-2 py-2">
                    {translateLabel("workspaceFileManager.modified")}
                  </th>
                  <th className="w-40 px-2 py-2 text-right">
                    {translateLabel("workspaceFileManager.actions")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {visibleEntries.map((entry) => (
                  <tr key={entry.path} className="hover:bg-slate-50">
                    <td className="min-w-0 px-3 py-2">
                      <button
                        type="button"
                        className="flex w-full min-w-0 items-center gap-2 text-left"
                        onClick={() => {
                          if (entry.is_dir) {
                            setCurrentPath(entry.path);
                            setPageIndex(0);
                          } else {
                            setPreviewPath(entry.path);
                          }
                        }}
                      >
                        <span className="shrink-0">
                          <EntryIcon entry={entry} />
                        </span>
                        <WorkspaceEntryName name={entry.name} />
                      </button>
                      <CompactTimestamp className="workspace-time-inline" value={entry.modified_at} locale={effectiveLocale}/>
                    </td>
                    <td className="truncate px-2 py-2 text-slate-500">
                      {entry.is_dir ? "-" : formatBytes(entry.size)}
                    </td>
                    <td className="workspace-time-column whitespace-nowrap px-2 py-2 text-xs text-slate-500"><CompactTimestamp value={entry.modified_at} locale={effectiveLocale}/></td>
                    <td className="px-2 py-2">
                      <div className="flex justify-end gap-1">
                        {!entry.is_dir && entry.previewable && (
                          <button
                            type="button"
                            className="cm-icon-button h-8 w-8"
                            title={translateLabel("workspaceFileManager.preview")}
                            onClick={() => setPreviewPath(entry.path)}
                          >
                            <Eye className="h-4 w-4" />
                          </button>
                        )}
                        {entry.downloadable && (
                          <button
                            type="button"
                            className="cm-icon-button h-8 w-8"
                            title={translateLabel("workspaceFileManager.download")}
                            onClick={() => handleDownload(entry)}
                            disabled={busyAction === `download:${entry.path}`}
                          >
                            <Download className="h-4 w-4" />
                          </button>
                        )}
                        {canWrite && (
                          <>
                            <button
                              type="button"
                              className="cm-icon-button h-8 w-8"
                              title={canEditEntry(entry) ? "编辑" : translateLabel("workspaceFileManager.rename")}
                              aria-label={`${canEditEntry(entry) ? "编辑" : "重命名"} ${entry.name}`}
                              onClick={() => canEditEntry(entry) ? setEditPath(entry.path) : handleRename(entry)}
                              disabled={busyAction === `rename:${entry.path}`}
                            >
                              {canEditEntry(entry) ? <FilePenLine className="h-4 w-4" /> : <TextCursorInput className="h-4 w-4" />}
                            </button>
                            <button
                              type="button"
                              className="cm-icon-button h-8 w-8 border-red-200 text-red-600 hover:border-red-300 hover:bg-red-50 hover:text-red-700"
                              title={translateLabel("workspaceFileManager.delete")}
                              onClick={() => handleDelete(entry)}
                              disabled={busyAction === `delete:${entry.path}`}
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {pageCount > 1 && (
              <div className="sticky bottom-0 flex items-center justify-between border-t border-slate-200 bg-white px-3 py-2 text-xs text-slate-600">
                <button
                  type="button"
                  className="rounded border border-slate-200 px-2 py-1 disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={visiblePageIndex === 0}
                  onClick={() => setPageIndex((value) => Math.max(0, value - 1))}
                >
                  {translateLabel("workspaceFileManager.previousPage")}
                </button>
                <span>
                  {translateLabel("workspaceFileManager.pageStatus", {current: visiblePageIndex + 1, total: pageCount})}
                </span>
                <button
                  type="button"
                  className="rounded border border-slate-200 px-2 py-1 disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={visiblePageIndex >= pageCount - 1}
                  onClick={() => setPageIndex((value) => Math.min(pageCount - 1, value + 1))}
                >
                  {translateLabel("workspaceFileManager.nextPage")}
                </button>
              </div>
            )}
            </>
          )}
        </div>

        {editPath && <WorkspaceTextEditor instanceId={instanceId} initialPath={editPath} service={service} canWrite={canWrite} onClose={()=>setEditPath(null)} onSaved={async()=>{setPreviewPath(null);await invalidateCurrentPath();await queryClient.invalidateQueries({queryKey:["workspace-preview",cacheKey]});await notifyMutation()}}/>}
        <PreviewPane
          path={previewPath}
          preview={preview}
          loading={previewQuery.isLoading}
          objectUrl={previewObjectUrl}
          onClose={() => setPreviewPath(null)}
          translateLabel={translateLabel}
        />
      </div>
    </section>
  );
}

function PreviewPane({
  path,
  preview,
  loading,
  objectUrl,
  onClose,
  translateLabel,
}: {
  path: string | null;
  preview?: WorkspacePreview;
  loading: boolean;
  objectUrl: string | null;
  onClose: () => void;
  translateLabel: (
    key: string,
    variables?: Record<string, string | number>,
  ) => string;
}) {
  if (!path) {
    return (
      <aside className="hidden border-t border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">
        {translateLabel("workspaceFileManager.preview")}
      </aside>
    );
  }

  return (
    <aside className="flex min-h-0 flex-1 flex-col overflow-hidden border-t border-slate-200 bg-slate-50">
      <div className="flex h-11 shrink-0 items-center justify-between gap-3 border-b border-slate-200 px-3">
        <div className="min-w-0 truncate text-sm font-medium text-slate-900">{fileName(path)}</div>
        <button
          type="button"
          className="cm-icon-button h-8 w-8"
          title={translateLabel("workspaceFileManager.close")}
          onClick={onClose}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto p-3">
        {loading ? (
          <div className="flex h-full items-center justify-center text-sm text-slate-500">
            {translateLabel("workspaceFileManager.loading")}
          </div>
        ) : preview?.kind === "text" ? (
          <pre className="min-w-0 max-w-full whitespace-pre-wrap break-words rounded-md border border-slate-200 bg-white p-3 font-mono text-xs leading-5 text-slate-800">
            {preview.text}
          </pre>
        ) : preview?.kind === "image" && objectUrl ? (
          <img
            src={objectUrl}
            alt={fileName(path)}
            className="mx-auto max-h-full max-w-full rounded-md border border-slate-200 bg-white object-contain"
          />
        ) : preview?.kind === "pdf" && objectUrl ? (
          <iframe
            title={fileName(path)}
            src={objectUrl}
            className="h-full min-h-[320px] w-full rounded-md border border-slate-200 bg-white"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-slate-500">
            {translateLabel("workspaceFileManager.downloadOnly")}
          </div>
        )}
      </div>
    </aside>
  );
}
