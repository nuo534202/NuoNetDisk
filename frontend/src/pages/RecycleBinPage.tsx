import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { recycleBinService } from "../services/recycleBin";
import type { File, Folder } from "../types";
import styles from "./RecycleBinPage.module.css";

function formatDateTime(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatExpiration(expiresAt: string): string {
  const now = Date.now();
  const expiry = new Date(expiresAt).getTime();
  const diffMs = expiry - now;
  if (diffMs <= 0) return "Expired";
  const totalHours = Math.floor(diffMs / (60 * 60 * 1000));
  const days = Math.floor(totalHours / 24);
  const hours = totalHours % 24;
  if (days > 0) return `${days}d ${hours}h remaining`;
  return `${totalHours}h remaining`;
}

function getExtension(name: string): string {
  const dot = name.lastIndexOf(".");
  return dot > 0 ? name.slice(dot).toLowerCase() : "";
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
  restore: "M4 12a8 8 0 1 0 2-6.18|M4 2v5h5",
  delete: "M3.5 5.5h13|M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5|M5.5 5.5l.73 10.93a1 1 0 0 0 1 .93h5.54a1 1 0 0 0 1-.93L14.5 5.5",
} as const;

type Tab = "files" | "folders";

export default function RecycleBinPage() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("files");
  const [files, setFiles] = useState<File[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);

  useEffect(() => {
    if (tab === "files") {
      recycleBinService.listFiles({ limit: 100 })
        .then((res) => setFiles(res.items))
        .catch(() => setFiles([]));
    } else {
      recycleBinService.listFolders({ limit: 100 })
        .then((res) => setFolders(res.items))
        .catch(() => setFolders([]));
    }
  }, [tab]);

  function reloadFiles() {
    recycleBinService.listFiles({ limit: 100 })
      .then((res) => setFiles(res.items))
      .catch(() => setFiles([]));
  }

  function reloadFolders() {
    recycleBinService.listFolders({ limit: 100 })
      .then((res) => setFolders(res.items))
      .catch(() => setFolders([]));
  }

  async function handleRestoreFile(fileId: string) {
    await recycleBinService.restoreFile(fileId);
    reloadFiles();
  }

  async function handlePermanentDeleteFile(fileId: string) {
    await recycleBinService.permanentDeleteFile(fileId);
    reloadFiles();
  }

  async function handleRestoreFolder(folderId: string) {
    await recycleBinService.restoreFolder(folderId);
    reloadFolders();
  }

  async function handlePermanentDeleteFolder(folderId: string) {
    await recycleBinService.permanentDeleteFolder(folderId);
    reloadFolders();
  }

  const displayItems = tab === "files" ? files : folders;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>Recycle Bin</span>
        <div className={styles.headerActions}>
          <Link to={`/${user?.user_hash}`} className={styles.linkBtn}>Back to files</Link>
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
          <div className={styles.gridTable}>
            <div className={styles.gridHeader}>
              <span></span>
              <span>Name</span>
              <span>Deleted at</span>
              <span>Expires in</span>
              <span>Actions</span>
            </div>

            {displayItems.map((item) => {
              const isFile = "size" in item;
              const f = item as File;
              const folder = item as Folder;

              return (
                <div key={item.id} className={styles.gridRow}>
                  <span className={styles.rowIcon}>
                    {isFile
                      ? <SvgIcon path={ICON_PATHS[getFileType(getExtension(item.name)) as keyof typeof ICON_PATHS] || ICON_PATHS.generic} />
                      : <SvgIcon path={ICON_PATHS.folder} />}
                  </span>
                  <span className={styles.rowName}>{item.name}</span>
                  <span className={styles.rowDate}>
                    {item.deleted_at ? formatDateTime(item.deleted_at) : "-"}
                  </span>
                  <span className={styles.rowExpiry}>
                    {item.expires_at ? formatExpiration(item.expires_at) : "-"}
                  </span>
                  <span className={styles.rowActions}>
                    <button
                      className={`${styles.actionBtn} ${styles.restoreBtn}`}
                      onClick={() => isFile ? handleRestoreFile(f.id) : handleRestoreFolder(folder.id)}
                      title="Restore"
                    >
                      <SvgIcon path={ICON_PATHS.restore} />
                    </button>
                    <button
                      className={`${styles.actionBtn} ${styles.deleteBtn}`}
                      onClick={() => isFile ? handlePermanentDeleteFile(f.id) : handlePermanentDeleteFolder(folder.id)}
                      title="Delete permanently"
                    >
                      <SvgIcon path={ICON_PATHS.delete} />
                    </button>
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
