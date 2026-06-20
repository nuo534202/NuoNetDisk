import { BrowserRouter, Routes, Route, Navigate, useParams, useNavigate } from "react-router-dom";
import { AuthProvider } from "./store/AuthContext";
import { useAuth } from "./store/useAuth";
import { folderService } from "./services/folders";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import DashboardPage from "./pages/DashboardPage";
import ProfilePage from "./pages/ProfilePage";
import RecycleBinPage from "./pages/RecycleBinPage";
import SharedPage from "./pages/SharedPage";
import AdminDashboardPage from "./pages/AdminDashboardPage";
import AdminUserManagementPage from "./pages/AdminUserManagementPage";
import AdminCreateAdminPage from "./pages/AdminCreateAdminPage";
import NotFoundPage from "./pages/NotFoundPage";
import { useEffect } from "react";
import type { ReactNode } from "react";

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, user } = useAuth();
  const location = window.location.pathname;

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (user?.is_admin && location !== "/profile") {
    return <Navigate to="/admin" replace />;
  }

  return <>{children}</>;
}

function AdminRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, user } = useAuth();

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!user?.is_admin) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}

function PublicRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, user } = useAuth();

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (isAuthenticated) {
    return <Navigate to={user?.is_admin ? "/admin" : "/"} replace />;
  }

  return <>{children}</>;
}

function HomeRedirect() {
  const { isAuthenticated, isLoading, user } = useAuth();
  if (isLoading) return <div>Loading...</div>;
  if (!isAuthenticated || !user) return <Navigate to="/login" replace />;
  if (user.is_admin) return <Navigate to="/admin" replace />;
  return <Navigate to={`/${user.user_hash}`} replace />;
}

function OldFolderRedirect() {
  const { folderId } = useParams<{ folderId: string }>();
  const { isAuthenticated, isLoading, user } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated || !user || !folderId) {
      navigate("/login", { replace: true });
      return;
    }
    folderService
      .get(folderId)
      .then((folder) => {
        navigate(`/${user.user_hash}/${encodeURIComponent(folder.name)}`, { replace: true });
      })
      .catch(() => {
        navigate(`/${user.user_hash}`, { replace: true });
      });
  }, [folderId, user, isAuthenticated, isLoading, navigate]);

  return <div>Redirecting...</div>;
}

function AppRoutes() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <PublicRoute>
            <LoginPage />
          </PublicRoute>
        }
      />
      <Route
        path="/register"
        element={
          <PublicRoute>
            <RegisterPage />
          </PublicRoute>
        }
      />
      <Route
        path="/admin"
        element={
          <AdminRoute>
            <AdminDashboardPage />
          </AdminRoute>
        }
      />
      <Route
        path="/admin/users"
        element={
          <AdminRoute>
            <AdminUserManagementPage />
          </AdminRoute>
        }
      />
      <Route
        path="/admin/create-admin"
        element={
          <AdminRoute>
            <AdminCreateAdminPage />
          </AdminRoute>
        }
      />
      <Route
        path="/recycle-bin"
        element={
          <ProtectedRoute>
            <RecycleBinPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/profile"
        element={
          <ProtectedRoute>
            <ProfilePage />
          </ProtectedRoute>
        }
      />
      <Route path="/s/:token" element={<SharedPage />} />
      <Route
        path="/folder/:folderId"
        element={
          <ProtectedRoute>
            <OldFolderRedirect />
          </ProtectedRoute>
        }
      />
      <Route path="/" element={<HomeRedirect />} />
      <Route
        path="/:userHash"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/:userHash/*"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
