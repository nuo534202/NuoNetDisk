import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminService } from "../services/admin";
import type { ErrorResponse } from "../types";
import styles from "./AdminDashboardPage.module.css";

export default function AdminCreateAdminPage() {
  const navigate = useNavigate();
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newPasswordConfirm, setNewPasswordConfirm] = useState("");
  const [creatingAdmin, setCreatingAdmin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  async function handleCreateAdmin(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    if (!newEmail.trim()) {
      setError("Email is required");
      return;
    }
    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    if (newPassword !== newPasswordConfirm) {
      setError("Passwords do not match");
      return;
    }

    setCreatingAdmin(true);
    try {
      await adminService.registerAdmin(newEmail.trim(), newPassword);
      setSuccess(true);
      setNewEmail("");
      setNewPassword("");
      setNewPasswordConfirm("");
    } catch (err) {
      const apiError = err as ErrorResponse;
      if (apiError?.error?.code === "DUPLICATE") {
        setError("Email is already in use");
      } else if (apiError?.error?.code === "FORBIDDEN") {
        setError("Only the super admin can create admin accounts");
      } else {
        setError(apiError?.error?.message || "Failed to create admin account");
      }
    } finally {
      setCreatingAdmin(false);
    }
  }

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.logo}>NuoNetDisk</span>
          <span className={styles.adminBadge}>Admin</span>
        </div>
        <div className={styles.headerActions}>
          <span
            className={styles.backBtn}
            onClick={() => navigate("/admin")}
          >
            Back
          </span>
        </div>
      </header>

      <div className={styles.content}>
        <div className={styles.centeredSection}>
          <div className={styles.section}>
            <h1 className={styles.createAdminTitle}>Create Admin Account</h1>
            <p className={styles.createAdminDesc}>Create a new admin account. The super admin account configured in the environment cannot be created here.</p>
            <form className={styles.createAdminForm} onSubmit={handleCreateAdmin}>
              {error && <div className={styles.formError}>{error}</div>}
              {success && <div className={styles.formSuccess}>Admin account created successfully.</div>}
              <div className={styles.formGroup}>
                <label className={styles.formLabel} htmlFor="admin-email">Email</label>
                <input
                  id="admin-email"
                  className={styles.formInput}
                  type="email"
                  placeholder="admin@example.com"
                  value={newEmail}
                  onChange={(e) => { setNewEmail(e.target.value); setSuccess(false); }}
                  autoComplete="off"
                />
              </div>
              <div className={styles.formGroup}>
                <label className={styles.formLabel} htmlFor="admin-password">Password</label>
                <input
                  id="admin-password"
                  className={styles.formInput}
                  type="password"
                  placeholder="At least 8 characters"
                  value={newPassword}
                  onChange={(e) => { setNewPassword(e.target.value); setSuccess(false); }}
                  autoComplete="off"
                />
              </div>
              <div className={styles.formGroup}>
                <label className={styles.formLabel} htmlFor="admin-password-confirm">Confirm Password</label>
                <input
                  id="admin-password-confirm"
                  className={styles.formInput}
                  type="password"
                  placeholder="Re-enter password"
                  value={newPasswordConfirm}
                  onChange={(e) => { setNewPasswordConfirm(e.target.value); setSuccess(false); }}
                  autoComplete="off"
                />
              </div>
              <button
                className={styles.createBtn}
                type="submit"
                disabled={creatingAdmin}
              >
                {creatingAdmin ? "Creating..." : "Create Admin Account"}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
