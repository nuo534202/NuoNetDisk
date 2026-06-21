import { useState, useEffect, useRef, useCallback, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { fileService } from "../services/files";
import { folderService } from "../services/folders";
import { getAccessToken } from "../services/api";
import type { File as AppFile, Folder } from "../types";
import styles from "./DashboardPage.module.css";

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function getFileType(ext: string): string {
  const imageExts = [".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".bmp", ".ico"];
  const videoExts = [".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm"];
  const audioExts = [".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a"];
  const archiveExts = [".zip", ".rar", ".tar", ".gz", ".7z", ".bz2"];
  if (imageExts.includes(ext)) return "image";
  if (videoExts.includes(ext)) return "video";
  if (audioExts.includes(ext)) return "audio";
  if (archiveExts.includes(ext)) return "archive";
  if ([".txt", ".md", ".log", ".rtf"].includes(ext)) return "text";
  if ([".pdf"].includes(ext)) return "pdf";
  if ([".doc", ".docx"].includes(ext)) return "word";
  if ([".xls", ".xlsx", ".csv"].includes(ext)) return "sheet";
  if ([".ppt", ".pptx"].includes(ext)) return "slide";
  return "generic";
}

function SvgIcon({ path, viewBox = "0 0 20 20" }: { path: string; viewBox?: string }) {
  return (
    <svg width="18" height="18" viewBox={viewBox} fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {path.split("|").map((d, i) => {
        const isCircle = d.startsWith("circle:");
        const isRect = d.startsWith("rect:");
        if (isCircle) {
          const [cx, cy, r] = d.replace("circle:", "").split(" ").map(Number);
          return <circle key={i} cx={cx} cy={cy} r={r} />;
        }
        if (isRect) {
          const [x, y, w, h, rx] = d.replace("rect:", "").split(" ").map(Number);
          return <rect key={i} x={x} y={y} width={w} height={h} rx={rx || 0} />;
        }
        return <path key={i} d={d} />;
      })}
    </svg>
  );
}

const ICON_PATHS = {
  folder: "M2.5 5.5h6l2-2h7a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1z",
  parentUp: "M10 17V3|M4 9l6-6 6 6",
  image: "rect:2.5 3.5 15 13 1.5|circle:7 8 1.5|M2.5 13.5l4-4 4 4|M10.5 11.5l2-2 5 4",
  video: "rect:2.5 3.5 15 13 1.5|M8.5 7.5l5 2.5-5 2.5v-5z",
  audio: "M10 17a3 3 0 1 0 0-6 3 3 0 0 0 0 6z|M13 3l-6 2v10",
  text: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5|M7 9.5h5|M7 12.5h5|M7 15.5h3",
  pdf: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5|M7.5 11V9a2 2 0 1 1 4 0v2|rect:7.5 11 4 3 0.6",
  word: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5|M7.5 14l1.5-5 1 3 1-3 1.5 5",
  sheet: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5|rect:7.5 10 5 5 0.5|M7.5 12.5h5|M10 10v5",
  slide: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5|M7.5 15V9.5h2V15|M10 15v-3h2v3|M12.5 15v-5.5h2V15",
  archive: "rect:2.5 3.5 15 13 1.5|M7 8.5h6|M7 11.5h6|M10 5.5v3",
  generic: "M4.5 3.5h7l5 5v8a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-12a1 1 0 0 1 1-1z|M11.5 3.5v5h5",
  details: "circle:10 10 7.5|M10 14v-4|M10 7v-.5",
  rename: "M16.5 3.5a2.12 2.12 0 1 1 3 3L7 19l-4 1 1-4L16.5 3.5z",
  move: "M4 10h9|M13 6l4 4-4 4",
  download: "M3 15v2a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-2|M10 3v10|M6 9l4 4 4-4",
  delete: "M3.5 5.5h13|M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5|M5.5 5.5l.73 10.93a1 1 0 0 0 1 .93h5.54a1 1 0 0 0 1-.93L14.5 5.5",
} as const;

interface BreadcrumbItem {
  id: string | null;
  name: string;
}

function getFileLocation(file: AppFile, crumbs: BreadcrumbItem[]): string {
  if (!file.parent_folder_id) return "Root";
  const path = crumbs.map((c) => c.name).join(" / ");
  return path || "Root";
}

export default function DashboardPage() {
  const { user, logout } = useAuth();
  const params = useParams();
  const folderPath = params["*"] || "";
  const pathSegments = folderPath ? folderPath.split("/").filter(Boolean) : [];
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [files, setFiles] = useState<AppFile[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([{ id: null, name: "Root" }]);
  const [currentFolderId, setCurrentFolderId] = useState<string | null>(null);
  const [currentFolderParentId, setCurrentFolderParentId] = useState<string | null>(null);
  const [resolving, setResolving] = useState(true);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [uploading, setUploading] = useState(false);

  const [renameTarget, setRenameTarget] = useState<{ type: "file" | "folder"; id: string; name: string } | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [showExtWarning, setShowExtWarning] = useState(false);
  const [pendingRenameValue, setPendingRenameValue] = useState("");

  const [moveTarget, setMoveTarget] = useState<{ type: "file" | "folder"; id: string; name: string } | null>(null);
  const [moveFolderStack, setMoveFolderStack] = useState<{ id: string | null; name: string }[]>([
    { id: null, name: "Root" },
  ]);
  const [moveFolderList, setMoveFolderList] = useState<Folder[]>([]);
  const [moveLoading, setMoveLoading] = useState(false);

  const [detailsFile, setDetailsFile] = useState<AppFile | null>(null);
  const [previewFile, setPreviewFile] = useState<AppFile | null>(null);

  const pendingNavRef = useRef<{ id: string; name: string } | null>(null);
  const currentFolderNameRef = useRef("");

  useEffect(() => {
    if (pendingNavRef.current) {
      const nav = pendingNavRef.current;
      pendingNavRef.current = null;
      setCurrentFolderId(nav.id);
      currentFolderNameRef.current = nav.name;
      setResolving(false);
      return;
    }

    if (pathSegments.length === 0) {
      setCurrentFolderId(null);
      currentFolderNameRef.current = "";
      setResolving(false);
      return;
    }

    setResolving(true);
    const pathStr = pathSegments.join("/");
    folderService
      .resolveByPath(pathStr)
      .then((result) => {
        setCurrentFolderId(result.folder.id);
        currentFolderNameRef.current = result.folder.name;
        setResolving(false);
      })
      .catch(() => {
        const hash = user?.user_hash;
        if (hash) {
          window.history.pushState({}, "", `/${hash}`);
          window.dispatchEvent(new PopStateEvent("popstate"));
        }
        setResolving(false);
      });
  }, [folderPath, user?.user_hash]);

  const reload = useCallback((fid: string | null) => {
    Promise.all([
      fileService.list({ folder_id: fid || undefined, limit: 100 }),
      folderService.list(fid || undefined),
    ])
      .then(([fileRes, folderRes]) => {
        setFiles(fileRes.items);
        setFolders(folderRes.items);
      })
      .catch(() => {
        setFiles([]);
        setFolders([]);
      });

    if (fid) {
      folderService
        .getAncestors(fid)
        .then((ancestors) => {
          setBreadcrumbs([
            { id: null, name: "Root" },
            ...ancestors.map((a) => ({ id: a.id, name: a.name })),
          ]);
        })
        .catch(() => setBreadcrumbs([{ id: null, name: "Root" }]));

      folderService
        .get(fid)
        .then((folder) => setCurrentFolderParentId(folder.parent_folder_id))
        .catch(() => setCurrentFolderParentId(null));
    } else {
      setBreadcrumbs([{ id: null, name: "Root" }]);
      setCurrentFolderParentId(null);
    }
  }, []);

  useEffect(() => {
    if (!resolving) {
      reload(currentFolderId);
    }
  }, [currentFolderId, resolving, reload]);

  function navigateToFolder(fid: string | null, fname?: string) {
    const hash = user?.user_hash;
    if (!hash) return;

    if (fid && fname) {
      pendingNavRef.current = { id: fid, name: fname };
      const newPath = pathSegments.length > 0
        ? [...pathSegments, encodeURIComponent(fname)].join("/")
        : encodeURIComponent(fname);
      navigate(`/${hash}/${newPath}`);
    } else {
      navigate(`/${hash}`);
    }
  }

  function navigateToParent(parentFid: string | null) {
    if (!parentFid || pathSegments.length <= 1) {
      navigateToFolder(null);
      return;
    }
    const parentPath = pathSegments.slice(0, -1).join("/");
    const parentCrumb = breadcrumbs.length > 1 ? breadcrumbs[breadcrumbs.length - 2] : null;
    if (parentCrumb && parentCrumb.id) {
      pendingNavRef.current = { id: parentCrumb.id, name: parentCrumb.name };
    }
    const hash = user?.user_hash;
    if (hash) {
      navigate(`/${hash}/${parentPath}`);
    }
  }

  async function handleCreateFolder(e: FormEvent) {
    e.preventDefault();
    if (!newFolderName.trim()) return;
    try {
      await folderService.create(newFolderName.trim(), currentFolderId);
      setNewFolderName("");
      setShowCreateDialog(false);
      reload(currentFolderId);
    } catch {
      // error handled silently
    }
  }

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      await fileService.upload(file, currentFolderId || undefined);
      reload(currentFolderId);
    } catch {
      // error handled silently
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  function handleDownload(fileId: string) {
    const url = fileService.downloadUrl(fileId);
    const a = document.createElement("a");
    a.href = url;
    a.click();
  }

  async function handleDeleteFile(fileId: string) {
    try {
      await fileService.delete(fileId);
      reload(currentFolderId);
    } catch {
      // error handled silently
    }
  }

  async function handleDeleteFolder(folderId: string) {
    try {
      await folderService.delete(folderId);
      reload(currentFolderId);
    } catch {
      // error handled silently
    }
  }

  function openRename(type: "file" | "folder", id: string, name: string) {
    setRenameTarget({ type, id, name });
    setRenameValue(name);
  }

  function getExtension(name: string): string {
    const dot = name.lastIndexOf(".");
    return dot > 0 ? name.slice(dot) : "";
  }

  async function performRename(id: string, type: "file" | "folder", newName: string) {
    try {
      if (type === "file") {
        await fileService.update(id, { name: newName });
      } else {
        await folderService.update(id, { name: newName });
      }
      setRenameTarget(null);
      setRenameValue("");
      reload(currentFolderId);

      if (type === "folder" && id === currentFolderId) {
        const hash = user?.user_hash;
        if (hash) {
          const newPath = [...pathSegments.slice(0, -1), encodeURIComponent(newName)].join("/");
          navigate(`/${hash}/${newPath}`);
        }
      }
    } catch {
      // error handled silently
    }
  }

  function handleRename(e: FormEvent) {
    e.preventDefault();
    if (!renameTarget || !renameValue.trim()) return;
    const newName = renameValue.trim();
    if (renameTarget.type === "file") {
      const oldExt = getExtension(renameTarget.name);
      const newExt = getExtension(newName);
      if (oldExt && newExt && oldExt !== newExt) {
        setPendingRenameValue(newName);
        setShowExtWarning(true);
        return;
      }
    }
    performRename(renameTarget.id, renameTarget.type, newName);
  }

  function confirmExtChange() {
    if (!renameTarget || !pendingRenameValue) return;
    performRename(renameTarget.id, renameTarget.type, pendingRenameValue);
    setShowExtWarning(false);
    setPendingRenameValue("");
  }

  function openMove(type: "file" | "folder", id: string, name: string) {
    setMoveTarget({ type, id, name });
    setMoveFolderStack([{ id: null, name: "Root" }]);
    loadMoveFolders(null);
  }

  function currentMoveFolderId(): string | null {
    return moveFolderStack[moveFolderStack.length - 1]?.id ?? null;
  }

  async function loadMoveFolders(parentId: string | null) {
    setMoveLoading(true);
    try {
      const res = await folderService.list(parentId || undefined);
      setMoveFolderList(res.items);
    } catch {
      setMoveFolderList([]);
    } finally {
      setMoveLoading(false);
    }
  }

  function moveNavigateToFolder(fid: string, name: string) {
    setMoveFolderStack((prev) => [...prev, { id: fid, name }]);
    loadMoveFolders(fid);
  }

  function moveNavigateUp() {
    if (moveFolderStack.length <= 1) return;
    const newStack = moveFolderStack.slice(0, -1);
    setMoveFolderStack(newStack);
    loadMoveFolders(newStack[newStack.length - 1]?.id ?? null);
  }

  async function handleMoveHere() {
    if (!moveTarget) return;
    const destFolderId = currentMoveFolderId();
    try {
      if (moveTarget.type === "file") {
        await fileService.update(moveTarget.id, { parent_folder_id: destFolderId });
      } else {
        await folderService.update(moveTarget.id, { parent_folder_id: destFolderId });
      }
      setMoveTarget(null);
      setMoveFolderStack([{ id: null, name: "Root" }]);
      setMoveFolderList([]);
      reload(currentFolderId);
    } catch {
      // error handled silently
    }
  }

  if (resolving) {
    return <div className={styles.container}><div className={styles.empty}><p>Loading...</p></div></div>;
  }

  const hasContent = folders.length > 0 || files.length > 0;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>NuoNetDisk</span>
        <div className={styles.headerActions}>
          {user?.is_admin && (
            <span className={styles.linkBtn} onClick={() => navigate('/admin')}>
              Admin Panel
            </span>
          )}
          <span className={styles.userInfo} onClick={() => navigate('/profile')}>
            {user?.avatar_url && (
              <img
                src={`${user.avatar_url}?token=${encodeURIComponent(getAccessToken() || "")}`}
                alt=""
                className={styles.userAvatar}
              />
            )}
            {user?.display_name}
          </span>
          <button className={styles.logoutBtn} onClick={logout}>Sign Out</button>
        </div>
      </header>

      <div className={styles.toolbar}>
        <button className={styles.toolbarBtn} onClick={() => setShowCreateDialog(true)}>
          + New Folder
        </button>
        <button className={styles.toolbarBtn} onClick={() => fileInputRef.current?.click()}>
          {uploading ? "Uploading..." : "Upload File"}
        </button>
        <a href="/recycle-bin" className={styles.recycleBinBtn}>Recycle Bin</a>
        <input
          ref={fileInputRef}
          className={styles.hiddenInput}
          type="file"
          onChange={handleUpload}
        />
      </div>

      <div className={styles.breadcrumb}>
        {breadcrumbs.map((item, i) => (
          <span key={item.id ?? "root"} style={{ display: "flex", alignItems: "center", gap: "0.4rem" }}>
            {i > 0 && <span className={styles.breadcrumbSep}>/</span>}
            {i === breadcrumbs.length - 1 ? (
              <span className={styles.breadcrumbCurrent}>{item.name}</span>
            ) : (
              <span className={styles.breadcrumbLink} onClick={() => {
                const targetPath = pathSegments.slice(0, i).join("/");
                const hash = user?.user_hash;
                if (!hash) return;
                if (item.id) {
                  pendingNavRef.current = { id: item.id, name: item.name };
                }
                navigate(targetPath ? `/${hash}/${targetPath}` : `/${hash}`);
              }}>
                {item.name}
              </span>
            )}
          </span>
        ))}
      </div>

      <div className={styles.content}>
        {!hasContent && !currentFolderId ? (
          <div className={styles.empty}>
            <div className={styles.emptyIcon}>&#128193;</div>
            <p>This folder is empty</p>
            <p style={{ fontSize: "0.85rem" }}>Upload a file or create a folder to get started</p>
          </div>
        ) : (
          <div className={styles.gridTable}>
            {currentFolderId && (
              <div className={styles.gridRow} onClick={() => navigateToParent(currentFolderParentId)}>
                <span className={styles.rowIcon}><SvgIcon path={ICON_PATHS.parentUp} /></span>
                <span className={styles.rowName} style={{ fontStyle: "italic", color: "var(--muted)" }}>..</span>
                <span className={styles.rowSize}></span>
                <span className={styles.rowDate}></span>
                <span className={styles.rowActions}></span>
              </div>
            )}

            {hasContent && (
              <>
                <div className={styles.gridHeader}>
                  <span></span>
                  <span>Name</span>
                  <span>Size</span>
                  <span>Date</span>
                  <span>Actions</span>
                </div>

                {folders.map((f) => (
                  <div key={f.id} className={styles.gridRow} onClick={() => navigateToFolder(f.id, f.name)}>
                    <span className={styles.rowIcon}><SvgIcon path={ICON_PATHS.folder} /></span>
                    <span className={styles.rowName}>{f.name}</span>
                    <span className={styles.rowSize}>-</span>
                    <span className={styles.rowDate}>{formatDate(f.created_at)}</span>
                    <span className={styles.rowActions}>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openRename("folder", f.id, f.name); }} title="Rename">
                        <SvgIcon path={ICON_PATHS.rename} />
                      </button>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openMove("folder", f.id, f.name); }} title="Move">
                        <SvgIcon path={ICON_PATHS.move} />
                      </button>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); handleDeleteFolder(f.id); }} title="Delete">
                        <SvgIcon path={ICON_PATHS.delete} />
                      </button>
                    </span>
                  </div>
                ))}

                {files.map((f) => (
                  <div key={f.id} className={styles.gridRow} onClick={() => setPreviewFile(f)}>
                    <span className={styles.rowIcon}>
                  <SvgIcon path={ICON_PATHS[getFileType(getExtension(f.name)) as keyof typeof ICON_PATHS] || ICON_PATHS.generic} />
                </span>
                    <span className={styles.rowName}>{f.name}</span>
                    <span className={styles.rowSize}>{formatSize(f.size)}</span>
                    <span className={styles.rowDate}>{formatDate(f.created_at)}</span>
                    <span className={styles.rowActions}>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); setDetailsFile(f); }} title="Details">
                        <SvgIcon path={ICON_PATHS.details} />
                      </button>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openRename("file", f.id, f.name); }} title="Rename">
                        <SvgIcon path={ICON_PATHS.rename} />
                      </button>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openMove("file", f.id, f.name); }} title="Move">
                        <SvgIcon path={ICON_PATHS.move} />
                      </button>
                      <button className={styles.downloadBtn} onClick={(e) => { e.stopPropagation(); handleDownload(f.id); }} title="Download">
                        <SvgIcon path={ICON_PATHS.download} />
                      </button>
                      <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); handleDeleteFile(f.id); }} title="Delete">
                        <SvgIcon path={ICON_PATHS.delete} />
                      </button>
                    </span>
                  </div>
                ))}
              </>
            )}
          </div>
        )}
      </div>

      {showCreateDialog && (
        <div className={styles.dialog} onClick={() => setShowCreateDialog(false)}>
          <div className={styles.dialogCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.dialogTitle}>New Folder</h2>
            <form onSubmit={handleCreateFolder}>
              <input
                className={styles.dialogInput}
                type="text"
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                placeholder="Folder Name"
                autoFocus
              />
              <div className={styles.dialogActions}>
                <button type="button" className={styles.dialogCancel} onClick={() => setShowCreateDialog(false)}>
                  Cancel
                </button>
                <button type="submit" className={styles.dialogConfirm}>
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {renameTarget && (
        <div className={styles.dialog} onClick={() => setRenameTarget(null)}>
          <div className={styles.dialogCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.dialogTitle}>Rename {renameTarget.type === "file" ? "file" : "folder"}</h2>
            <form onSubmit={handleRename}>
              <input
                className={styles.dialogInput}
                type="text"
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                placeholder="New name"
                autoFocus
              />
              <div className={styles.dialogActions}>
                <button type="button" className={styles.dialogCancel} onClick={() => setRenameTarget(null)}>
                  Cancel
                </button>
                <button type="submit" className={styles.dialogConfirm}>
                  Rename
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showExtWarning && (
        <div className={styles.dialog}>
          <div className={styles.dialogCard}>
            <h2 className={styles.dialogTitle}>Change file extension?</h2>
            <p style={{ marginBottom: "1rem", color: "var(--muted)", fontSize: "0.9rem", lineHeight: "1.5" }}>
              You are about to change the extension from <strong>{getExtension(renameTarget?.name || "")}</strong> to <strong>{getExtension(pendingRenameValue)}</strong>.
              This may make the file unusable. Are you sure?
            </p>
            <div className={styles.dialogActions}>
              <button type="button" className={styles.dialogCancel} onClick={() => { setShowExtWarning(false); setPendingRenameValue(""); }}>
                Cancel
              </button>
              <button type="button" className={styles.dialogConfirm} onClick={confirmExtChange}>
                Confirm
              </button>
            </div>
          </div>
        </div>
      )}

      {moveTarget && (
        <div className={styles.dialog} onClick={() => setMoveTarget(null)}>
          <div className={styles.moveDialogCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.dialogTitle}>
              Move &ldquo;{moveTarget.name}&rdquo;
            </h2>

            <div className={styles.moveBreadcrumb}>
              <span
                className={styles.moveBreadcrumbLink}
                onClick={() => {
                  setMoveFolderStack([{ id: null, name: "Root" }]);
                  loadMoveFolders(null);
                }}
              >
                Root
              </span>
              {moveFolderStack.slice(1).map((item, i) => (
                <span key={item.id ?? "root"} style={{ display: "flex", alignItems: "center", gap: "0.25rem" }}>
                  <span className={styles.breadcrumbSep}>/</span>
                  {i === moveFolderStack.length - 2 ? (
                    <span className={styles.moveBreadcrumbCurrent}>{item.name}</span>
                  ) : (
                    <span
                      className={styles.moveBreadcrumbLink}
                      onClick={() => {
                        const newStack = moveFolderStack.slice(0, i + 2);
                        setMoveFolderStack(newStack);
                        loadMoveFolders(newStack[newStack.length - 1]?.id ?? null);
                      }}
                    >
                      {item.name}
                    </span>
                  )}
                </span>
              ))}
            </div>

            <div className={styles.moveFolderList}>
              {moveFolderStack.length > 1 && (
                <div className={styles.moveFolderItem} onClick={moveNavigateUp}>
                  <span className={styles.moveFolderIcon}>&#128281;</span>
                  <span>..</span>
                </div>
              )}
              {moveLoading ? (
                <div className={styles.moveEmpty}>Loading...</div>
              ) : moveFolderList.length === 0 ? (
                <div className={styles.moveEmpty}>No subfolders</div>
              ) : (
                moveFolderList.map((folder) => (
                  <div
                    key={folder.id}
                    className={styles.moveFolderItem}
                    onClick={() => moveNavigateToFolder(folder.id, folder.name)}
                  >
                    <span className={styles.moveFolderIcon}>&#128193;</span>
                    <span>{folder.name}</span>
                  </div>
                ))
              )}
            </div>

            <div className={styles.dialogActions}>
              <button
                type="button"
                className={styles.dialogCancel}
                onClick={() => {
                  setMoveTarget(null);
                  setMoveFolderStack([{ id: null, name: "Root" }]);
                  setMoveFolderList([]);
                }}
              >
                Cancel
              </button>
              <button type="button" className={styles.dialogConfirm} onClick={handleMoveHere}>
                Move here
              </button>
            </div>
          </div>
        </div>
      )}

      {detailsFile && (
        <div className={styles.dialog} onClick={() => setDetailsFile(null)}>
          <div className={styles.detailsCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.dialogTitle}>File details</h2>
            <table className={styles.detailsTable}>
              <tbody>
                <tr>
                  <td className={styles.detailsLabel}>Name</td>
                  <td className={styles.detailsValue}>{detailsFile.name}</td>
                </tr>
                <tr>
                  <td className={styles.detailsLabel}>Size</td>
                  <td className={styles.detailsValue}>{formatSize(detailsFile.size)}</td>
                </tr>
                <tr>
                  <td className={styles.detailsLabel}>Location</td>
                  <td className={styles.detailsValue}>{getFileLocation(detailsFile, breadcrumbs)}</td>
                </tr>

                <tr>
                  <td className={styles.detailsLabel}>Created</td>
                  <td className={styles.detailsValue}>{formatDate(detailsFile.created_at)}</td>
                </tr>
                <tr>
                  <td className={styles.detailsLabel}>Updated</td>
                  <td className={styles.detailsValue}>{formatDate(detailsFile.updated_at)}</td>
                </tr>
              </tbody>
            </table>
            <div className={styles.dialogActions}>
              <button type="button" className={styles.dialogConfirm} onClick={() => setDetailsFile(null)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {previewFile && (
        <div className={styles.dialog} onClick={() => setPreviewFile(null)}>
          <div className={styles.previewCard} onClick={(e) => e.stopPropagation()}>
            <div className={styles.previewHeader}>
              <div className={styles.previewInfo}>
                <h2 className={styles.dialogTitle}>{previewFile.name}</h2>
                <span className={styles.previewMeta}>{formatSize(previewFile.size)}</span>
              </div>
              <button
                className={styles.actionBtn}
                onClick={(e) => { e.stopPropagation(); handleDownload(previewFile.id); }}
                title="Download"
              >
                <SvgIcon path={ICON_PATHS.download} />
              </button>
            </div>
            <div className={styles.previewBody}>
              {previewFile.mime_type.startsWith("image/") ? (
                <img
                  src={fileService.previewUrl(previewFile.id)}
                  alt={previewFile.name}
                  className={styles.previewImage}
                />
              ) : previewFile.mime_type.startsWith("video/") ? (
                <video controls className={styles.previewMedia} src={fileService.previewUrl(previewFile.id)}>
                  Your browser does not support video preview.
                </video>
              ) : previewFile.mime_type.startsWith("audio/") ? (
                <audio controls className={styles.previewAudio} src={fileService.previewUrl(previewFile.id)}>
                  Your browser does not support audio preview.
                </audio>
              ) : previewFile.mime_type === "application/pdf" || previewFile.mime_type.startsWith("text/") ? (
                <iframe
                  src={fileService.previewUrl(previewFile.id)}
                  className={styles.previewIframe}
                  title="File preview"
                />
              ) : (
                <div className={styles.previewUnsupported}>
                  <p>Preview not available for this file type.</p>
                  <p className={styles.previewUnsupportedHint}>
                    <button
                      className={styles.dialogConfirm}
                      onClick={() => handleDownload(previewFile.id)}
                    >
                      Download file
                    </button>
                  </p>
                </div>
              )}
            </div>
            <div className={styles.dialogActions}>
              <button
                type="button"
                className={styles.dialogConfirm}
                onClick={() => setPreviewFile(null)}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}