import { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { userService } from "../services/user";
import styles from "./ProfilePage.module.css";

export default function ProfilePage() {
  const { user, logout, updateUser } = useAuth();
  const [displayName, setDisplayName] = useState(user?.display_name ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!displayName.trim()) return;
    setSaving(true);
    setError(null);
    setSuccess(false);
    try {
      const updated = await userService.updateMe({ display_name: displayName.trim() });
      updateUser(updated);
      setSuccess(true);
    } catch {
      setError("Failed to update profile");
    } finally {
      setSaving(false);
    }
  }

  const hasChanges = displayName !== user?.display_name;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <span className={styles.logo}>Profile</span>
        <div className={styles.headerActions}>
          <Link to="/" className={styles.linkBtn}>Back to files</Link>
          <button className={styles.linkBtn} onClick={logout}>Sign out</button>
        </div>
      </header>

      <div className={styles.content}>
        <div className={styles.section}>
          <h1 className={styles.sectionTitle}>Account settings</h1>

          <div className={styles.field}>
            <span className={styles.label}>Email</span>
            <div className={styles.readonlyValue}>{user?.email}</div>
          </div>

          <form onSubmit={handleSave}>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="displayName">
                Display name
              </label>
              <input
                id="displayName"
                className={styles.input}
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Your display name"
              />
            </div>

            <button
              className={styles.saveBtn}
              type="submit"
              disabled={saving || !hasChanges}
            >
              {saving ? "Saving..." : "Save changes"}
            </button>

            {error && <p className={styles.error}>{error}</p>}
            {success && <p className={styles.success}>Profile updated</p>}
          </form>
        </div>
      </div>
    </div>
  );
}
