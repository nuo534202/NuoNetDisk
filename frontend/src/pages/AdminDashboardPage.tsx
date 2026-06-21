import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { adminService } from "../services/admin";
import { getAccessToken } from "../services/api";
import type { DashboardData } from "../types";
import styles from "./AdminDashboardPage.module.css";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

const REFRESH_OPTIONS = [
  { value: 0, label: "Off" },
  { value: 10, label: "10s" },
  { value: 30, label: "30s" },
  { value: 60, label: "1m" },
  { value: 300, label: "5m" },
];
const REFRESH_STORAGE_KEY = "admin_dashboard_refresh_seconds";
const DEFAULT_REFRESH_SECONDS = 30;

function loadRefreshSeconds(): number {
  const raw = localStorage.getItem(REFRESH_STORAGE_KEY);
  if (raw === null) return DEFAULT_REFRESH_SECONDS;
  const parsed = Number(raw);
  return REFRESH_OPTIONS.some((o) => o.value === parsed) ? parsed : DEFAULT_REFRESH_SECONDS;
}

export default function AdminDashboardPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [dashboard, setDashboard] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshSeconds, setRefreshSeconds] = useState<number>(loadRefreshSeconds);

  const fetchDashboard = useCallback(async () => {
    try {
      const data = await adminService.getDashboard();
      setDashboard(data);
      setError(null);
    } catch {
      setError("Failed to load admin dashboard");
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchDashboard().finally(() => setLoading(false));
  }, [fetchDashboard]);

  // Adjustable auto-refresh. Default 30s balances data freshness against
  // server load; the admin can disable it (0) or pick a different cadence.
  useEffect(() => {
    if (refreshSeconds <= 0) return;
    const timer = setInterval(() => {
      fetchDashboard();
    }, refreshSeconds * 1000);
    return () => clearInterval(timer);
  }, [refreshSeconds, fetchDashboard]);

  function handleRefreshChange(value: string) {
    const seconds = Number(value);
    setRefreshSeconds(seconds);
    localStorage.setItem(REFRESH_STORAGE_KEY, String(seconds));
  }

  if (loading) {
    return <div className={styles.container}><div className={styles.loading}>Loading admin panel...</div></div>;
  }

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.logo}>NuoNetDisk</span>
          <span className={styles.adminBadge}>Admin</span>
        </div>
        <div className={styles.headerActions}>
          <div className={styles.refreshControl}>
            <span className={styles.refreshLabel}>Refresh</span>
            <select
              className={styles.refreshSelect}
              value={refreshSeconds}
              onChange={(e) => handleRefreshChange(e.target.value)}
              title="Auto-refresh interval"
            >
              {REFRESH_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <span className={styles.profileBtn} onClick={() => navigate("/profile")}>
            {user?.avatar_url && (
              <img
                src={`${user.avatar_url}?token=${encodeURIComponent(getAccessToken() || "")}`}
                alt=""
                className={styles.userAvatar}
              />
            )}
            {user?.display_name}
          </span>
          <span className={styles.navBtn} onClick={() => navigate("/admin/users")}>
            User Management
          </span>
          {user?.is_super_admin && (
            <span className={styles.navBtn} onClick={() => navigate("/admin/create-admin")}>
              Create Admin
            </span>
          )}
          <span className={styles.signOutBtn} onClick={logout}>Sign Out</span>
        </div>
      </header>

      <div className={styles.content}>
        {error && <div className={styles.error}>{error}</div>}

        <div className={styles.grid}>
          <div className={styles.card}>
            <div className={styles.cardTitle}>Project Info</div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Name</span>
              <span className={styles.infoValue}>{dashboard?.project_info.name}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Version</span>
              <span className={styles.infoValue}>{dashboard?.project_info.version}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Uptime</span>
              <span className={styles.infoValue}>{dashboard?.project_info.uptime}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Go Version</span>
              <span className={styles.infoValue}>{dashboard?.project_info.go_version}</span>
            </div>
          </div>

          <div className={styles.card}>
            <div className={styles.cardTitle}>Services</div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Database</span>
              <span className={styles.infoValue}>
                <span className={`${styles.statusDot} ${dashboard?.services.database ? styles.statusUp : styles.statusDown}`} />
                {dashboard?.services.database ? "Healthy" : "Unhealthy"}
              </span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>MinIO</span>
              <span className={styles.infoValue}>
                <span className={`${styles.statusDot} ${dashboard?.services.minio ? styles.statusUp : styles.statusDown}`} />
                {dashboard?.services.minio ? "Healthy" : "Unhealthy"}
              </span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Backend</span>
              <span className={styles.infoValue}>
                <span className={`${styles.statusDot} ${dashboard?.services.backend ? styles.statusUp : styles.statusDown}`} />
                {dashboard?.services.backend ? "Running" : "Down"}
              </span>
            </div>
          </div>

          <div className={styles.card}>
            <div className={styles.cardTitle}>System Resources</div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Goroutines</span>
              <span className={styles.infoValue}>{dashboard?.system.num_goroutine}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Allocated Memory</span>
              <span className={styles.infoValue}>{dashboard?.system.allocated_mb} MB</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Total Allocated</span>
              <span className={styles.infoValue}>{dashboard?.system.total_allocated_mb} MB</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>GC Cycles</span>
              <span className={styles.infoValue}>{dashboard?.system.num_gc}</span>
            </div>
          </div>

          <div className={styles.card}>
            <div className={styles.cardTitle}>Storage Stats</div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Users</span>
              <span className={styles.infoValue}>{dashboard?.stats.total_users}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Files</span>
              <span className={styles.infoValue}>{dashboard?.stats.total_files}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Folders</span>
              <span className={styles.infoValue}>{dashboard?.stats.total_folders}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Storage Used</span>
              <span className={styles.infoValue}>{dashboard?.stats.total_storage_bytes ? formatBytes(dashboard.stats.total_storage_bytes) : "0 B"}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Active Shares</span>
              <span className={styles.infoValue}>{dashboard?.stats.active_shares}</span>
            </div>
            <div className={styles.infoRow}>
              <span className={styles.infoLabel}>Recycle Bin</span>
              <span className={styles.infoValue}>{dashboard?.stats.recycle_bin_count} items</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
