import { useState, useRef } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../store/useAuth";
import { userService } from "../services/user";
import { getAccessToken } from "../services/api";
import styles from "./ProfilePage.module.css";

const GENDER_OPTIONS = [
  { value: "", label: "Prefer not to say" },
  { value: "male", label: "Male" },
  { value: "female", label: "Female" },
];

export default function ProfilePage() {
  const { user, logout, updateUser } = useAuth();
  const [displayName, setDisplayName] = useState(user?.display_name ?? "");
  const [bio, setBio] = useState(user?.bio ?? "");
  const [gender, setGender] = useState(user?.gender ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [avatarRev, setAvatarRev] = useState(() => Date.now());
  const fileInputRef = useRef<HTMLInputElement>(null);

  const currentAvatarSrc = avatarPreview
    || (user?.avatar_url
      ? `${user.avatar_url}?token=${encodeURIComponent(getAccessToken() || "")}&_=${encodeURIComponent(user.updated_at || "")}&r=${avatarRev}`
      : "");

  function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    const reader = new FileReader();
    reader.onload = () => setAvatarPreview(reader.result as string);
    reader.readAsDataURL(file);
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!displayName.trim()) return;
    setSaving(true);
    setError(null);
    setSuccess(false);
    try {
      let updated = user;
      if (avatarFile) {
        updated = await userService.uploadAvatar(avatarFile);
      }
      if (
        displayName.trim() !== updated?.display_name ||
        bio.trim() !== updated?.bio ||
        gender !== updated?.gender
      ) {
        updated = await userService.updateMe({
          display_name: displayName.trim(),
          bio: bio.trim(),
          gender,
        });
      } else if (!avatarFile) {
        updated = await userService.updateMe({
          display_name: displayName.trim(),
          bio: bio.trim(),
          gender,
        });
      }
      if (updated) {
        updateUser(updated);
      }
      setSuccess(true);
      setAvatarFile(null);
      setAvatarPreview(null);
      setAvatarRev(Date.now());
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    } catch {
      setError("Failed to update profile");
    } finally {
      setSaving(false);
    }
  }

  const hasChanges =
    displayName !== user?.display_name ||
    bio !== user?.bio ||
    gender !== user?.gender ||
    avatarFile !== null;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.logo}>NuoNetDisk</span>
          {user?.is_admin && <span className={styles.adminBadge}>Admin</span>}
        </div>
        <div className={styles.headerActions}>
          <Link to={user?.is_admin ? "/admin" : `/${user?.user_hash}`} className={styles.linkBtn}>Back</Link>
          <button className={styles.signOutBtn} onClick={logout}>Sign Out</button>
        </div>
      </header>

      <div className={styles.content}>
        <div className={styles.section}>
          <h1 className={styles.sectionTitle}>Account Settings</h1>

          <div className={styles.field}>
            <span className={styles.label}>Email</span>
            <div className={styles.readonlyValue}>{user?.email}</div>
          </div>

          <div className={styles.avatarSection}>
            {currentAvatarSrc && (
              <div className={styles.avatarPreview}>
                <img src={currentAvatarSrc} alt="Avatar" className={styles.avatarImg} />
              </div>
            )}
            <div className={styles.field}>
              <label className={styles.label} htmlFor="avatarUpload">
                Avatar
              </label>
              <input
                id="avatarUpload"
                ref={fileInputRef}
                className={styles.fileInput}
                type="file"
                accept="image/*"
                onChange={handleAvatarChange}
              />
            </div>
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

            <div className={styles.field}>
              <label className={styles.label} htmlFor="bio">
                Bio
              </label>
              <textarea
                id="bio"
                className={styles.textarea}
                rows={3}
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                placeholder="Tell us about yourself"
              />
            </div>

            <div className={styles.field}>
              <label className={styles.label} htmlFor="gender">
                Gender
              </label>
              <select
                id="gender"
                className={styles.select}
                value={gender}
                onChange={(e) => setGender(e.target.value)}
              >
                {GENDER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <button
              className={styles.saveBtn}
              type="submit"
              disabled={saving || !hasChanges}
            >
              {saving ? "Saving..." : "Save Changes"}
            </button>

            {error && <p className={styles.error}>{error}</p>}
            {success && <p className={styles.success}>Profile updated</p>}
          </form>
        </div>
      </div>
    </div>
  );
}
