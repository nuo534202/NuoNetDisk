import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { recycleBinService } from "../services/recycleBin";
import type { File, Folder } from "../types";
import styles from "./RecycleBinPage.module.css";

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

type Tab = "files" | "folders";

export default function RecycleBinPage() {
  const [tab, setTab] = useState<Tab>("files");
  const [files, setFiles] = useState<File[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);

  async function loadFiles() {
    const res = await recycleBinService.listFiles({ limit: 100 });
    setFiles(res.items);
  }

  async function loadFolders() {
    const res = await recycleBinService.listFolders({ limit: 100 });
    setFolders(res.items);
  }

  useEffect(() => {
    if (tab === "files") loadFiles();
    else loadFolders();
  }, [tab]);

  async function handleRestoreFile(fileId: string) {
    await recycleBinService.restoreFile(fileId);
    loadFiles();
  }

  async function handlePermanentDeleteFile(fileId: string) {
    await recycleBinService.permanentDeleteFile(fileId);
    loadFiles();
  }

  async function handleRestoreFolder(folderId: string) {
    await recycleBinService.restoreFolder(folderId);
    loadFolders();
  }

  async function handlePermanentDeleteFolder(folderId: string) {
    await recycleBinService.permanentDeleteFolder(folderId);
    loadFolders();
  }

  const displayItems = tab === "files" ? files : folders;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>Recycle Bin</span>
        <div className={styles.headerActions}>
          <Link to="/" className={styles.linkBtn}>Back to files</Link>
        </div>
      </header>

      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${tab === "files" ? styles.tabActive : ""}`}
          onClick={() => setTab("files")}
        >
          Files
        </button>
        <button
          className={`${styles.tab} ${tab === "folders" ? styles.tabActive : ""}`}
          onClick={() => setTab("folders")}
        >
          Folders
        </button>
      </div>

      <div className={styles.content}>
        {displayItems.length === 0 ? (
          <div className={styles.empty}>
            <div className={styles.emptyIcon}>&#128466;</div>
            <p>No deleted {tab} found</p>
          </div>
        ) : (
          <>
            <div className={styles.gridHeader}>
              <span></span>
              <span>Name</span>
              <span>Deleted</span>
              <span></span>
            </div>

            {displayItems.map((item) => {
              const isFile = "size" in item;
              const f = item as File;
              const folder = item as Folder;

              return (
                <div key={item.id} className={styles.gridRow}>
                  <span className={styles.rowIcon}>{isFile ? "\u{1F4C4}" : "\u{1F4C1}"}</span>
                  <span className={styles.rowName}>{item.name}</span>
                  <span className={styles.rowDate}>{formatDate(item.updated_at)}</span>
                  <span className={styles.rowActions}>
                    <button
                      className={`${styles.actionBtn} ${styles.restoreBtn}`}
                      onClick={() => isFile ? handleRestoreFile(f.id) : handleRestoreFolder(folder.id)}
                      title="Restore"
                    >
                      Restore
                    </button>
                    <button
                      className={`${styles.actionBtn} ${styles.deleteBtn}`}
                      onClick={() => isFile ? handlePermanentDeleteFile(f.id) : handlePermanentDeleteFolder(folder.id)}
                      title="Delete permanently"
                    >
                      Delete
                    </button>
                  </span>
                </div>
              );
            })}
          </>
        )}
      </div>
    </div>
  );
}
