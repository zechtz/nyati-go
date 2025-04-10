import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";

// Auth context
import { useAuth } from "./contexts/AuthContext";
import Login from "./Login";
import Register from "./Register";
import MainLayout from "./layout/MainLayout";
import Dashboard from "./dashboard";
import NotFound from "./NotFound";
import ConfigsPage from "./App";
import BlueprintList from "./blueprints/BlueprintList";
import BlueprintForm from "./blueprints/BlueprintForm";
import BlueprintUse from "./blueprints/BlueprintUse";
import SandboxSimulator from "./sandbox/SandboxSimulator";
import EnvironmentsPage from "./env/Environments";

// Protected route component
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

const AppRouter = () => {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />

        {/* Protected routes with MainLayout */}
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <MainLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="blueprints" element={<BlueprintList />} />
          <Route path="blueprints/new" element={<BlueprintForm />} />
          <Route path="blueprints/edit/:id" element={<BlueprintForm />} />
          <Route path="blueprints/use/:id" element={<BlueprintUse />} />
          <Route path="sandbox" element={<SandboxSimulator />} />
          <Route path="configs" element={<ConfigsPage />} />
          <Route path="configs/:configPath" element={<ConfigsPage />} />
          <Route path="environments" element={<EnvironmentsPage />} />

          {/* Add placeholder routes for other sections */}
          <Route
            path="deployments"
            element={<ComingSoon title="Deployments" />}
          />
          <Route path="tasks" element={<ComingSoon title="Tasks" />} />
          <Route path="settings" element={<ComingSoon title="Settings" />} />
          <Route path="users" element={<ComingSoon title="Manage Users" />} />
          <Route
            path="environments"
            element={<ComingSoon title="Manage Environments" />}
          />

          {/* 404 for unknown paths within the authenticated area */}
          <Route path="*" element={<NotFound />} />
        </Route>

        {/* Fallback route */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>

      <ToastContainer
        position="top-right"
        autoClose={3000}
        hideProgressBar={false}
        newestOnTop
        closeOnClick
        rtl={false}
        pauseOnFocusLoss
        draggable
        pauseOnHover
      />
    </BrowserRouter>
  );
};

// Simple placeholder component for routes that aren't implemented yet
const ComingSoon = ({ title }: { title: string }) => (
  <div className="flex flex-col items-center justify-center py-20">
    <h2 className="text-2xl font-bold mb-4">{title}</h2>
    <p className="text-gray-500 mb-6">This feature is coming soon!</p>
    <div className="w-24 h-1 bg-primary-500 rounded-full mb-6"></div>
    <p className="text-gray-600 max-w-md text-center">
      We're working hard to bring you this functionality. Check back soon for
      updates.
    </p>
  </div>
);

export default AppRouter;
