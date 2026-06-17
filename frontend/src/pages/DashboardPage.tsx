import { useState, useEffect, useRef, useCallback, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { fileService } from "../services/files";
import { folderService } from "../services/folders";
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

interface BreadcrumbItem {
  id: string | null;
  name: string;
}

export default function DashboardPage() {
  const { user, logout } = useAuth();
  const { folderId } = useParams<{ folderId: string }>();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [files, setFiles] = useState<AppFile[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([{ id: null, name: "Root" }]);
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

  const currentFolderId = folderId || null;

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

    const breadcrumbPromise = fid
      ? folderService.get(fid).then(
          (folder) => [
            { id: null, name: "Root" },
            { id: folder.id, name: folder.name },
          ],
        )
      : Promise.resolve([{ id: null, name: "Root" }]);
    breadcrumbPromise.then(setBreadcrumbs).catch(() => setBreadcrumbs([{ id: null, name: "Root" }]));
  }, []);

  useEffect(() => {
    reload(currentFolderId);
  }, [currentFolderId, reload]);

  function navigateToFolder(fid: string | null) {
    const path = fid ? `/folder/${fid}` : "/";
    window.history.pushState({}, "", path);
    window.dispatchEvent(new PopStateEvent("popstate"));
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

  const hasContent = folders.length > 0 || files.length > 0;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>NuoNetDisk</span>
        <div className={styles.headerActions}>
          <span className={styles.userInfo}>{user?.display_name}</span>
          <Link to="/recycle-bin" className={styles.linkBtn}>Recycle bin</Link>
          <Link to="/profile" className={styles.linkBtn}>Profile</Link>
          <button className={styles.logoutBtn} onClick={logout}>Sign out</button>
        </div>
      </header>

      <div className={styles.toolbar}>
        <button className={styles.toolbarBtn} onClick={() => setShowCreateDialog(true)}>
          + New folder
        </button>
        <button className={styles.toolbarBtn} onClick={() => fileInputRef.current?.click()}>
          {uploading ? "Uploading..." : "Upload file"}
        </button>
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
              <span className={styles.breadcrumbLink} onClick={() => navigateToFolder(item.id)}>
                {item.name}
              </span>
            )}
          </span>
        ))}
      </div>

      <div className={styles.content}>
        {!hasContent ? (
          <div className={styles.empty}>
            <div className={styles.emptyIcon}>&#128193;</div>
            <p>This folder is empty</p>
            <p style={{ fontSize: "0.85rem" }}>Upload a file or create a folder to get started</p>
          </div>
        ) : (
          <>
            <div className={styles.gridHeader}>
              <span></span>
              <span>Name</span>
              <span>Size</span>
              <span>Date</span>
              <span></span>
            </div>

            {folders.map((f) => (
              <div key={f.id} className={styles.gridRow} onClick={() => navigateToFolder(f.id)}>
                <span className={styles.rowIcon}>&#128193;</span>
                <span className={styles.rowName}>{f.name}</span>
                <span className={styles.rowSize}>-</span>
                <span className={styles.rowDate}>{formatDate(f.created_at)}</span>
                <span className={styles.rowActions}>
                  <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openRename("folder", f.id, f.name); }} title="Rename">
                    &#9998;
                  </button>
                  <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openMove("folder", f.id, f.name); }} title="Move">
                    &#8594;
                  </button>
                  <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); handleDeleteFolder(f.id); }} title="Delete">
                    &#128465;
                  </button>
                </span>
              </div>
            ))}

            {files.map((f) => (
              <div key={f.id} className={styles.gridRow}>
                <span className={styles.rowIcon}>&#128196;</span>
                <span className={styles.rowName}>{f.name}</span>
                <span className={styles.rowSize}>{formatSize(f.size)}</span>
                <span className={styles.rowDate}>{formatDate(f.created_at)}</span>
                <span className={styles.rowActions}>
                  <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openRename("file", f.id, f.name); }} title="Rename">
                    &#9998;
                  </button>
                  <button className={styles.actionBtn} onClick={(e) => { e.stopPropagation(); openMove("file", f.id, f.name); }} title="Move">
                    &#8594;
                  </button>
                  <button className={styles.downloadBtn} onClick={(e) => { e.stopPropagation(); handleDownload(f.id); }} title="Download">
                    &#8595;
                  </button>
                  <button className={styles.actionBtn} onClick={() => handleDeleteFile(f.id)} title="Delete">
                    &#128465;
                  </button>
                </span>
              </div>
            ))}
          </>
        )}
      </div>

      {showCreateDialog && (
        <div className={styles.dialog} onClick={() => setShowCreateDialog(false)}>
          <div className={styles.dialogCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.dialogTitle}>New folder</h2>
            <form onSubmit={handleCreateFolder}>
              <input
                className={styles.dialogInput}
                type="text"
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                placeholder="Folder name"
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
    </div>
  );
}
