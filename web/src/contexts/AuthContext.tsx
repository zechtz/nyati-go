import React, { createContext, useState, useContext, useEffect } from "react";
import axios from "axios";
import { toast } from "react-toastify";

interface User {
  id: number;
  email: string;
  created_at: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<boolean>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType>({
  isAuthenticated: false,
  user: null,
  loading: true,
  login: async () => false,
  logout: () => {},
});

export const useAuth = () => useContext(AuthContext);

interface AuthProviderProps {
  children: React.ReactNode;
}

const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Check if user is already logged in
    const checkAuthStatus = async () => {
      const token = localStorage.getItem("authToken");

      if (!token) {
        setLoading(false);
        return;
      }

      try {
        // Set the auth header
        axios.defaults.headers.common["Authorization"] = `Bearer ${token}`;

        // Verify token by making a request to a protected endpoint
        const response = await axios.get("/api/configs");

        // If we get here, the token is valid
        setIsAuthenticated(true);

        // Optionally fetch user data - would need a /api/me endpoint
        // const userResponse = await axios.get("/api/me");
        // setUser(userResponse.data);
      } catch (error) {
        // Token is invalid
        localStorage.removeItem("authToken");
        delete axios.defaults.headers.common["Authorization"];
      } finally {
        setLoading(false);
      }
    };

    checkAuthStatus();
  }, []);

  // Set up an axios interceptor to handle token expiration
  useEffect(() => {
    const interceptor = axios.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        // If the error is 401 Unauthorized and we haven't already tried to refresh the token
        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          isAuthenticated
        ) {
          originalRequest._retry = true;

          try {
            // Try to refresh the token
            const response = await axios.post("/api/refresh-token");
            const { token } = response.data;

            // Update the token in localStorage and axios headers
            localStorage.setItem("authToken", token);
            axios.defaults.headers.common["Authorization"] = `Bearer ${token}`;

            // Retry the original request
            return axios(originalRequest);
          } catch (refreshError) {
            // If refreshing the token fails, log the user out
            localStorage.removeItem("authToken");
            delete axios.defaults.headers.common["Authorization"];
            setIsAuthenticated(false);
            setUser(null);
            toast.error("Session expired. Please log in again.");

            return Promise.reject(refreshError);
          }
        }

        return Promise.reject(error);
      },
    );

    // Clean up the interceptor when the component unmounts
    return () => {
      axios.interceptors.response.eject(interceptor);
    };
  }, [isAuthenticated]);

  const login = async (email: string, password: string): Promise<boolean> => {
    try {
      const response = await axios.post("/api/login", { email, password });
      const { token, user } = response.data;

      // Save the token to localStorage
      localStorage.setItem("authToken", token);

      // Set the auth header for all future requests
      axios.defaults.headers.common["Authorization"] = `Bearer ${token}`;

      // Update the auth state
      setIsAuthenticated(true);
      setUser(user);

      return true;
    } catch (error) {
      return false;
    }
  };

  const logout = () => {
    // Send logout request to backend (optional)
    axios.post("/api/logout").catch(() => {
      // Even if the request fails, continue with local logout
    });

    // Remove token from localStorage
    localStorage.removeItem("authToken");

    // Remove auth header
    delete axios.defaults.headers.common["Authorization"];

    // Update auth state
    setIsAuthenticated(false);
    setUser(null);
  };

  const value = {
    isAuthenticated,
    user,
    loading,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export default AuthProvider;
