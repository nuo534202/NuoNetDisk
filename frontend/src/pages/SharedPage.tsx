import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { shareService } from "../services/shares";
import styles from "./SharedPage.module.css";

export default function SharedPage() {
  const { token } = useParams<{ token: string }>();
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    shareService.accessByToken(token)
      .then((res) => {
        setData(res as Record<string, unknown>);
        setLoading(false);
      })
      .catch(() => {
        setError("Share link not found or has expired");
        setLoading(false);
      });
  }, [token]);

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        {loading && (
          <>
            <h1 className={styles.title}>Loading...</h1>
            <p className={styles.subtitle}>Fetching shared resource</p>
          </>
        )}

        {error && (
          <>
            <h1 className={styles.title}>Not found</h1>
            <p className={styles.error}>{error}</p>
            <p className={styles.subtitle}>
              <Link to="/login">Sign in to your account</Link>
            </p>
          </>
        )}

        {data && !loading && !error && (
          <>
            <h1 className={styles.title}>Shared with you</h1>
            <p className={styles.detail}>
              {"name" in data ? (data.name as string) : "Shared resource"}
            </p>
            <p className={styles.subtitle}>
              <Link to="/login">Sign in to download</Link>
            </p>
          </>
        )}
      </div>
    </div>
  );
}
