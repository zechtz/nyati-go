import { useState, useEffect, useRef } from "react";
import { Outlet, useNavigate, NavLink, useLocation } from "react-router-dom";
import {
  Search,
  User,
  LogOut,
  Menu,
  X,
  ChartColumn,
  ChevronDown,
  ChevronRight,
  DatabaseZap,
  Plus,
  Settings,
  Settings2,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "react-toastify";
import { ConfigEntry, ConfigState } from "../App";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

// Animation durations in ms
const ANIMATION_DURATION = 300;

const MainLayout = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [isMobile, setIsMobile] = useState(false);
  const [isOnline, setIsOnline] = useState(true);

  // Sidebar state
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isBlueprintsOpen, setIsBlueprintsOpen] = useState(false);
  const [newConfigPath, setNewConfigPath] = useState("");
  const [, setConfigs] = useState<ConfigEntry[]>([]);
  const [, setConfigStates] = useState<{
    [key: string]: ConfigState;
  }>({});

  // Animation refs for menu containers
  const settingsMenuRef = useRef<HTMLDivElement>(null);
  const blueprintsMenuRef = useRef<HTMLDivElement>(null);

  // Handle responsive behavior
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 768) {
        setIsMobile(true);
        setSidebarOpen(false);
      } else {
        setIsMobile(false);
        setSidebarOpen(true);
      }
    };

    // Initial check
    handleResize();

    // Add event listener
    window.addEventListener("resize", handleResize);

    // Check online status
    const handleOnlineStatus = () => {
      setIsOnline(navigator.onLine);
    };

    window.addEventListener("online", handleOnlineStatus);
    window.addEventListener("offline", handleOnlineStatus);

    // Clean up
    return () => {
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("online", handleOnlineStatus);
      window.removeEventListener("offline", handleOnlineStatus);
    };
  }, []);

  // Automatically open settings or deployments section if on a related page
  useEffect(() => {
    if (
      location.pathname.startsWith("/settings") ||
      location.pathname.startsWith("/tasks") ||
      location.pathname.startsWith("/users") ||
      location.pathname.startsWith("/environments")
    ) {
      setIsSettingsOpen(true);
    }

    if (location.pathname.startsWith("/deployments")) {
      setIsBlueprintsOpen(true);
    }
  }, [location.pathname]);

  // Set heights for animation
  useEffect(() => {
    if (settingsMenuRef.current) {
      settingsMenuRef.current.style.maxHeight = isSettingsOpen
        ? `${settingsMenuRef.current.scrollHeight}px`
        : "0px";
    }
    if (blueprintsMenuRef.current) {
      blueprintsMenuRef.current.style.maxHeight = isBlueprintsOpen
        ? `${blueprintsMenuRef.current.scrollHeight}px`
        : "0px";
    }
  }, [isSettingsOpen, isBlueprintsOpen]);

  const handleLogout = () => {
    logout();
    toast.success("Logged out successfully!");
    navigate("/login");
  };

  const toggleSidebar = () => {
    setSidebarOpen(!sidebarOpen);
  };

  const toggleSettings = () => {
    setIsSettingsOpen(!isSettingsOpen);
  };

  const toggleBlueprints = () => {
    setIsBlueprintsOpen(!isBlueprintsOpen);
  };

  const addConfig = () => {
    if (newConfigPath) {
      const newConfig: ConfigEntry = {
        id: 0,
        name: "",
        description: "",
        path: newConfigPath,
        status: "DRAFT", // Default status for new configs
        user_id: user?.id || 1,
      };
      setConfigs((prev) => {
        const updatedConfigs = [...prev, newConfig];
        setConfigStates((prevStates) => ({
          ...prevStates,
          [newConfig.path]: {
            selectedHost: "all",
            selectedTask: "none",
            tasks: [],
            hosts: [],
          },
        }));
        return updatedConfigs;
      });
      setNewConfigPath("");
      toast.success("Config added successfully!");
      navigate("/configs");
    } else {
      toast.error("Please enter a config path.");
    }
  };

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Sidebar */}
      <aside
        className={`${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        } transform fixed inset-y-0 left-0 z-20 w-64 bg-secondary-500 text-white transition-transform duration-300 ease-in-out md:relative md:translate-x-0 flex flex-col h-full`}
      >
        {/* Logo section at top */}
        <div className="p-5 border-b border-gray-700 flex items-center justify-between bg-primary-500">
          <h1 className="text-2xl font-inter">Nyativ1.2.1</h1>
          {isMobile && (
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleSidebar}
              className="text-white md:hidden"
            >
              <X className="h-6 w-6" />
            </Button>
          )}
        </div>

        <nav className="flex-1 p-4 space-y-2 overflow-y-auto">
          <NavLink
            to="/dashboard"
            className={({ isActive }) =>
              `flex items-center p-2 rounded ${
                isActive ? "bg-primary-500" : "hover:bg-primary-600"
              }`
            }
            end
          >
            <ChartColumn className="h-5 w-5" />
            <span className="ml-2">Dashboard</span>
          </NavLink>

          <div>
            <button
              onClick={toggleBlueprints}
              className={`flex items-center p-2 rounded ${
                location.pathname.startsWith("/deployments")
                  ? "bg-primary-500"
                  : "hover:bg-primary-600"
              } w-full text-left transition-colors duration-200`}
            >
              <DatabaseZap className="h-5 w-5" />
              <span className="ml-2">Deployments</span>
              <div className="ml-auto">
                {isBlueprintsOpen ? (
                  <ChevronDown className="h-5 w-5 transition-transform duration-200" />
                ) : (
                  <ChevronRight className="h-5 w-5 transition-transform duration-200" />
                )}
              </div>
            </button>
            <div
              ref={blueprintsMenuRef}
              className="overflow-hidden transition-all ease-in-out pl-6 space-y-1"
              style={{
                maxHeight: "0",
                opacity: isBlueprintsOpen ? 1 : 0,
                transitionDuration: `${ANIMATION_DURATION}ms`,
              }}
            >
              <NavLink
                to="/deployments"
                className={({ isActive }) =>
                  `block p-2 rounded ${
                    isActive ? "bg-primary-500" : "hover:bg-primary-600"
                  } transition-colors duration-200`
                }
              >
                List Deployments
              </NavLink>
            </div>
          </div>

          <div>
            <button
              onClick={toggleSettings}
              className={`flex items-center p-2 rounded ${
                ["/settings", "/tasks", "/users", "/environments"].some(
                  (path) => location.pathname.startsWith(path),
                )
                  ? "bg-primary-500"
                  : "hover:bg-primary-600"
              } w-full text-left transition-colors duration-200`}
            >
              <Settings className="h-5 w-5" />
              <span className="ml-2">Settings</span>
              <div className="ml-auto">
                {isSettingsOpen ? (
                  <ChevronDown className="h-5 w-5 transition-transform duration-200" />
                ) : (
                  <ChevronRight className="h-5 w-5 transition-transform duration-200" />
                )}
              </div>
            </button>
            <div
              ref={settingsMenuRef}
              className="overflow-hidden transition-all ease-in-out pl-6 space-y-1"
              style={{
                maxHeight: "0",
                opacity: isSettingsOpen ? 1 : 0,
                transitionDuration: `${ANIMATION_DURATION}ms`,
              }}
            >
              <NavLink
                to="/settings"
                className={({ isActive }) =>
                  `block p-2 rounded ${
                    isActive ? "bg-primary-500" : "hover:bg-primary-600"
                  } transition-colors duration-200`
                }
                end
              >
                View Resource Usage
              </NavLink>
              <NavLink
                to="/tasks"
                className={({ isActive }) =>
                  `block p-2 rounded ${
                    isActive ? "bg-primary-500" : "hover:bg-primary-600"
                  } transition-colors duration-200`
                }
              >
                Manage Tasks
              </NavLink>
              <NavLink
                to="/users"
                className={({ isActive }) =>
                  `block p-2 rounded ${
                    isActive ? "bg-primary-500" : "hover:bg-primary-600"
                  } transition-colors duration-200`
                }
              >
                Manage Users
              </NavLink>
              <NavLink
                to="/environments"
                className={({ isActive }) =>
                  `block p-2 rounded ${
                    isActive ? "bg-primary-500" : "hover:bg-primary-600"
                  } transition-colors duration-200`
                }
              >
                Manage Environments
              </NavLink>
            </div>
          </div>
          <NavLink
            to="/configs"
            className={({ isActive }) =>
              `flex items-center p-2 rounded ${
                isActive ? "bg-primary-500" : "hover:bg-primary-600"
              } transition-colors duration-200`
            }
          >
            <Settings2 className="h-5 w-5" />
            <span className="ml-2">Manage Configs</span>
          </NavLink>
        </nav>

        {/* Config creation and status section at bottom */}
        <div className="mt-auto border-t border-gray-700">
          <div className="p-4 space-y-2">
            <Input
              placeholder="Config Path (e.g., nyati.live.yml)"
              value={newConfigPath}
              onChange={(e) => setNewConfigPath(e.target.value)}
              className="bg-secondary-600 text-white border-secondary-400 placeholder:text-gray-400"
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  addConfig();
                }
              }}
            />
            <Button
              className="w-full bg-cyan-500 hover:bg-secondary-600 transition-colors duration-200"
              onClick={addConfig}
            >
              <Plus className="h-5 w-5 mr-2" />
              Create Config
            </Button>
          </div>

          {/* Status indicator */}
          <div className="p-4 flex items-center">
            <div
              className={`w-3 h-3 ${
                isOnline ? "bg-green-500" : "bg-red-500"
              } rounded-full mr-2`}
            ></div>
            <span className="text-sm">
              Status: {isOnline ? "Online" : "Offline"}
            </span>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top header */}
        <header className="shadow-sm z-10 bg-white">
          <div className="flex justify-between items-center p-4">
            <div className="flex items-center">
              {isMobile && (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={toggleSidebar}
                  className="mr-2 md:hidden"
                >
                  <Menu className="h-6 w-6" />
                </Button>
              )}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
                <Input className="pl-10 w-64" placeholder="Search..." />
              </div>
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="flex items-center space-x-2">
                  <Avatar>
                    <AvatarImage
                      src="https://github.com/shadcn.png"
                      alt="User"
                    />
                    <AvatarFallback>
                      {user?.email ? user.email[0].toUpperCase() : "U"}
                    </AvatarFallback>
                  </Avatar>
                  <span className="hidden md:inline">{user?.email}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem>
                  <User className="mr-2 h-4 w-4" />
                  <span>Profile</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleLogout}>
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>Logout</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-6">
          {/* Backdrop for mobile */}
          {sidebarOpen && isMobile && (
            <div
              className="fixed inset-0 bg-black bg-opacity-50 z-10 md:hidden"
              onClick={() => setSidebarOpen(false)}
            ></div>
          )}

          {/* Dynamic content rendered here */}
          <Outlet />
        </main>
      </div>
    </div>
  );
};

export default MainLayout;
