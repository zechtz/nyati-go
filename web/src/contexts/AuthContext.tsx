import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import axios from "axios";
import { User } from "../App";

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  register: (email: string, password: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  token: null,
  isAuthenticated: false,
  login: async () => {},
  logout: () => {},
  register: async () => {},
});

export const useAuth = () => useContext(AuthContext);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  // Initialize auth state from localStorage
  useEffect(() => {
    const storedToken = localStorage.getItem("NYATI_TOKEN");
    const storedUser = localStorage.getItem("NYATI_USER");

    if (storedToken && storedUser) {
      setToken(storedToken);
      setUser(JSON.parse(storedUser));
      setIsAuthenticated(true);

      // Set axios default headers
      axios.defaults.headers.common["Authorization"] = `Bearer ${storedToken}`;
    }

    setIsLoading(false);
  }, []);

  // Setup axios interceptor to handle token refresh
  useEffect(() => {
    const interceptor = axios.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        // If the error is 401 and we haven't tried to refresh the token yet
        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          token
        ) {
          originalRequest._retry = true;

          try {
            const response = await axios.post("/api/refresh-token");
            const newToken = response.data.token;

            // Update token in state and localStorage
            setToken(newToken);
            localStorage.setItem("NYATI_TOKEN", newToken);

            // Update axios headers
            axios.defaults.headers.common["Authorization"] =
              `Bearer ${newToken}`;

            // Retry the original request
            return axios(originalRequest);
          } catch (refreshError) {
            // If refresh fails, log the user out
            logout();
            return Promise.reject(refreshError);
          }
        }

        return Promise.reject(error);
      },
    );

    return () => {
      axios.interceptors.response.eject(interceptor);
    };
  }, [token]);

  const login = async (email: string, password: string) => {
    try {
      const response = await axios.post("/api/login", { email, password });
      const { token, user } = response.data;

      // Store token and user in localStorage
      localStorage.setItem("NYATI_TOKEN", token);
      localStorage.setItem("NYATI_USER", JSON.stringify(user));

      // Update state
      setToken(token);
      setUser(user);
      setIsAuthenticated(true);

      // Set authorization header for future requests
      axios.defaults.headers.common["Authorization"] = `Bearer ${token}`;
    } catch (error) {
      console.error("Login failed:", error);
      throw error;
    }
  };

  const logout = () => {
    // Clear localStorage
    localStorage.removeItem("NYATI_TOKEN");
    localStorage.removeItem("NYATI_USER");

    // Clear state
    setToken(null);
    setUser(null);
    setIsAuthenticated(false);

    // Remove authorization header
    delete axios.defaults.headers.common["Authorization"];

    // Call logout API (optional, since JWT is stateless)
    axios.post("/api/logout").catch((error) => {
      console.error("Logout API call failed:", error);
    });
  };

  const register = async (email: string, password: string) => {
    try {
      await axios.post("/api/register", { email, password });
      // After registration, log the user in
      await login(email, password);
    } catch (error) {
      console.error("Registration failed:", error);
      throw error;
    }
  };

  if (isLoading) {
    return <div>Loading authentication...</div>;
  }

  return (
    <AuthContext.Provider
      value={{ user, token, isAuthenticated, login, logout, register }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export default AuthProvider;
