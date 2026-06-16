import { Link } from "react-router-dom";
import styles from "./NotFoundPage.module.css";

export default function NotFoundPage() {
  return (
    <div className={styles.container}>
      <h1 className={styles.title}>404 &mdash; Page Not Found</h1>
      <p className={styles.message}>The page you are looking for does not exist.</p>
      <Link className={styles.link} to="/">Go Home</Link>
    </div>
  );
}
