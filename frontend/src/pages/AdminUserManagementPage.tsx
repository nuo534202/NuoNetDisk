import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { adminService } from "../services/admin";
import { getAccessToken } from "../services/api";
import type { AdminUser } from "../types";
import styles from "./AdminDashboardPage.module.css";

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function genderLabel(g: string): string {
  if (g === "male") return "Male";
  if (g === "female") return "Female";
  return "";
}

export default function AdminUserManagementPage() {
  const navigate = useNavigate();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [totalUsers, setTotalUsers] = useState(0);
  const [userPage, setUserPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const firstFetchDone = useRef(false);

  const [searchQ, setSearchQ] = useState("");
  const [roleFilter, setRoleFilter] = useState("");
  const [genderFilter, setGenderFilter] = useState("");
  const [joinedFrom, setJoinedFrom] = useState("");
  const [joinedTo, setJoinedTo] = useState("");

  const pageSize = 20;

  const fetchUsers = useCallback(async () => {
    try {
      const res = await adminService.listUsers({
        offset: userPage * pageSize,
        limit: pageSize,
        q: searchQ,
        role_filter: roleFilter,
        gender_filter: genderFilter,
        date_from: joinedFrom,
        date_to: joinedTo,
      });
      setUsers(res.items);
      setTotalUsers(res.total);
      setError(null);
    } catch {
      setError("Failed to load users");
    }
  }, [userPage, searchQ, roleFilter, genderFilter, joinedFrom, joinedTo]);

  useEffect(() => {
    fetchUsers().finally(() => {
      if (!firstFetchDone.current) {
        firstFetchDone.current = true;
        setLoading(false);
      }
    });
  }, [fetchUsers]);

  function handleFilterChange(setter: (v: string) => void) {
    return (value: string) => {
      setter(value);
      setUserPage(0);
    };
  }

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.logo}>NuoNetDisk</span>
          <span className={styles.adminBadge}>Admin</span>
        </div>
        <div className={styles.headerActions}>
          <span className={styles.backBtn} onClick={() => navigate("/admin")}>Back</span>
        </div>
      </header>

      <div className={styles.content}>
        {error && <div className={styles.error}>{error}</div>}

        <div className={styles.section}>
          <h1 className={styles.createAdminTitle}>User Management</h1>

          {loading ? (
            <div className={styles.loading}>Loading...</div>
          ) : (
            <>
              <div className={styles.filterRow}>
                <div className={styles.filterGroup}>
                  <span className={styles.filterLabel}>Search</span>
                  <input
                    className={styles.filterInput}
                    type="text"
                    placeholder="Email or name..."
                    value={searchQ}
                    onChange={(e) => handleFilterChange(setSearchQ)(e.target.value)}
                  />
                </div>
                <div className={styles.filterGroup}>
                  <span className={styles.filterLabel}>Role</span>
                  <select
                    className={styles.filterSelect}
                    value={roleFilter}
                    onChange={(e) => handleFilterChange(setRoleFilter)(e.target.value)}
                  >
                    <option value="">All</option>
                    <option value="admin">Admin</option>
                    <option value="user">User</option>
                  </select>
                </div>
                <div className={styles.filterGroup}>
                  <span className={styles.filterLabel}>Gender</span>
                  <select
                    className={styles.filterSelect}
                    value={genderFilter}
                    onChange={(e) => handleFilterChange(setGenderFilter)(e.target.value)}
                  >
                    <option value="">All</option>
                    <option value="male">Male</option>
                    <option value="female">Female</option>
                    <option value="unspecified">Prefer not to say</option>
                  </select>
                </div>
                <div className={styles.filterGroup}>
                  <span className={styles.filterLabel}>Joined From</span>
                  <input
                    data-testid="joined-from"
                    className={styles.filterInput}
                    type="date"
                    value={joinedFrom}
                    onChange={(e) => handleFilterChange(setJoinedFrom)(e.target.value)}
                  />
                </div>
                <div className={styles.filterGroup}>
                  <span className={styles.filterLabel}>Joined To</span>
                  <input
                    data-testid="joined-to"
                    className={styles.filterInput}
                    type="date"
                    value={joinedTo}
                    onChange={(e) => handleFilterChange(setJoinedTo)(e.target.value)}
                  />
                </div>
              </div>

              <table className={styles.usersTable}>
                <thead>
                  <tr>
                    <th>Avatar</th>
                    <th>Email</th>
                    <th>Display Name</th>
                    <th>Bio</th>
                    <th>Gender</th>
                    <th>Role</th>
                    <th>Joined</th>
                  </tr>
                </thead>
                <tbody>
                  {users.length === 0 ? (
                    <tr><td colSpan={7} style={{ padding: "1rem", textAlign: "center", color: "var(--muted)" }}>No users found</td></tr>
                  ) : (
                    users.map((u) => (
                      <tr key={u.id}>
                        <td>
                          {u.avatar_url ? (
                            <img src={`${u.avatar_url}?token=${encodeURIComponent(getAccessToken() || "")}`} alt="" className={styles.userAvatar} />
                          ) : (
                            <span className={styles.noneText}>None</span>
                          )}
                        </td>
                        <td>
                          {u.email}
                          {u.is_super_admin && <span className={styles.superAdminBadge}>Super Admin</span>}
                        </td>
                        <td>{u.display_name}</td>
                        <td className={styles.bioCell}>{u.bio || <span className={styles.noneText}>None</span>}</td>
                        <td>{genderLabel(u.gender) || <span className={styles.noneText}>None</span>}</td>
                        <td>
                          <span className={u.is_admin ? styles.adminTag : styles.userTag}>
                            {u.is_admin ? "Admin" : "User"}
                          </span>
                        </td>
                        <td>{formatDate(u.created_at)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
              {totalUsers > pageSize && (
                <div className={styles.pagination}>
                  <button className={styles.pageBtn} disabled={userPage === 0} onClick={() => setUserPage((p) => Math.max(0, p - 1))}>
                    Previous
                  </button>
                  <span className={styles.pageInfo}>
                    Page {userPage + 1} of {Math.ceil(totalUsers / pageSize)}
                  </span>
                  <button className={styles.pageBtn} disabled={(userPage + 1) * pageSize >= totalUsers} onClick={() => setUserPage((p) => p + 1)}>
                    Next
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
