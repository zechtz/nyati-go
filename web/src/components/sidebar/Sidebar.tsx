import {
  ChartColumn,
  ChevronDown,
  ChevronRight,
  DatabaseZap,
  Plus,
  Settings,
  Monitor,
  Activity,
  Database,
  Heart,
  FileText,
} from "lucide-react";
import { Link } from "react-router-dom";
import { Button } from "../ui/button";
import { useState } from "react";
import { ConfigEntry, ConfigState } from "@/App";
import { toast } from "react-toastify";

const Sidebar = () => {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isBlueprintsOpen, setIsBlueprintsOpen] = useState(false);
  const [isSystemOpen, setIsSystemOpen] = useState(true);

  // Debug logging
  console.log("Sidebar rendering, isSystemOpen:", isSystemOpen);
  const [newConfigPath, setNewConfigPath] = useState("");
  const [, setConfigs] = useState<ConfigEntry[]>([]);

  const [, setConfigStates] = useState<{
    [key: string]: ConfigState;
  }>({});

  const addConfig = () => {
    if (newConfigPath) {
      const newConfig: ConfigEntry = {
        id: "",
        name: "",
        description: "",
        path: newConfigPath,
        status: "DRAFT", // Default status for new configs
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
    } else {
      toast.error("Please enter a config path.");
    }
  };

  return (
    <div className="w-64 bg-secondary-500 text-white flex flex-col pt-16">
      <nav className="flex-1 p-4 space-y-2">
        <Link
          to="/"
          className="flex items-center p-2 rounded hover:bg-primary-600"
        >
          <ChartColumn className="h-5 w-5" />
          <span className="ml-2">Dashboard</span>
        </Link>
        <div>
          <button
            onClick={() => setIsBlueprintsOpen(!isBlueprintsOpen)}
            className="flex items-center p-2 rounded hover:bg-primary-600 w-full text-left"
          >
            <DatabaseZap className="h-5 w-5" />
            <span className="ml-2">Blueprints</span>
            {isBlueprintsOpen ? (
              <ChevronDown className="ml-auto h-5 w-5" />
            ) : (
              <ChevronRight className="ml-auto h-5 w-5" />
            )}
          </button>
          {isBlueprintsOpen && (
            <div className="pl-6 space-y-1">
              {/* Placeholder for future submenu items */}
              <Link
                to="/blueprints/list"
                className="block p-2 rounded hover:bg-primary-600"
              >
                List Blueprints
              </Link>
            </div>
          )}
        </div>

        {/* System Monitoring Section */}
        <div style={{ border: "1px solid red", margin: "4px" }}>
          <button
            onClick={() => setIsSystemOpen(!isSystemOpen)}
            className="flex items-center p-2 rounded hover:bg-primary-600 w-full text-left"
            style={{ backgroundColor: "orange" }}
          >
            <Monitor className="h-5 w-5" />
            <span className="ml-2">System DEBUG</span>
            {isSystemOpen ? (
              <ChevronDown className="ml-auto h-5 w-5" />
            ) : (
              <ChevronRight className="ml-auto h-5 w-5" />
            )}
          </button>
          {isSystemOpen && (
            <div className="pl-6 space-y-1">
              <Link
                to="/system/overview"
                className="flex items-center p-2 rounded hover:bg-primary-600"
              >
                <Activity className="h-4 w-4 mr-2" />
                Overview
              </Link>
              <Link
                to="/system/health"
                className="flex items-center p-2 rounded hover:bg-primary-600"
              >
                <Heart className="h-4 w-4 mr-2" />
                Health Status
              </Link>
              <Link
                to="/system/database"
                className="flex items-center p-2 rounded hover:bg-primary-600"
              >
                <Database className="h-4 w-4 mr-2" />
                Database Metrics
              </Link>
              <Link
                to="/system/settings"
                className="flex items-center p-2 rounded hover:bg-primary-600"
              >
                <Settings className="h-4 w-4 mr-2" />
                System Settings
              </Link>
              <Link
                to="/system/logs"
                className="flex items-center p-2 rounded hover:bg-primary-600"
              >
                <FileText className="h-4 w-4 mr-2" />
                System Logs
              </Link>
            </div>
          )}
        </div>

        <div>
          <button
            onClick={() => setIsSettingsOpen(!isSettingsOpen)}
            className="flex items-center p-2 rounded hover:bg-primary-600 w-full text-left"
          >
            <Monitor className="h-5 w-5" />
            <span className="ml-2">System DEBUG</span>
            {isSettingsOpen ? (
              <ChevronDown className="ml-auto h-5 w-5" />
            ) : (
              <ChevronRight className="ml-auto h-5 w-5" />
            )}
          </button>
          {isSettingsOpen && (
            <div className="pl-6 space-y-1">
              <Link
                to="/settings/resource-usage"
                className="block p-2 rounded hover:bg-primary-600"
              >
                View Resource Usage
              </Link>
              <Link
                to="/settings/installation"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Installation
              </Link>
              <Link
                to="/settings/users"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Users
              </Link>
              <Link
                to="/settings/machine-templates"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Machine Templates
              </Link>
              <Link
                to="/settings/environments"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Environments
              </Link>
              <Link
                to="/settings/images"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Images
              </Link>
            </div>
          )}
        </div>

        <div>
          <button
            onClick={() => setIsSettingsOpen(!isSettingsOpen)}
            className="flex items-center p-2 rounded hover:bg-primary-600 w-full text-left"
          >
            <Settings className="h-5 w-5" />
            <span className="ml-2">Settings</span>
            {isSettingsOpen ? (
              <ChevronDown className="ml-auto h-5 w-5" />
            ) : (
              <ChevronRight className="ml-auto h-5 w-5" />
            )}
          </button>
          {isSettingsOpen && (
            <div className="pl-6 space-y-1">
              <Link
                to="/settings/resource-usage"
                className="block p-2 rounded hover:bg-primary-600"
              >
                View Resource Usage
              </Link>
              <Link
                to="/settings/installation"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Installation
              </Link>
              <Link
                to="/settings/users"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Users
              </Link>
              <Link
                to="/settings/machine-templates"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Machine Templates
              </Link>
              <Link
                to="/settings/environments"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Environments
              </Link>
              <Link
                to="/settings/images"
                className="block p-2 rounded hover:bg-primary-600"
              >
                Manage Images
              </Link>
            </div>
          )}
        </div>
        <Link
          to="/configs"
          className="flex items-center p-2 rounded bg-primary-600"
        >
          <span className="ml-2">Manage Configs</span>
        </Link>
      </nav>
      <Button
        className="m-4 bg-cyan-500 hover:bg-secondary-600"
        onClick={addConfig}
      >
        <Plus className="h-5 w-5 mr-2" />
        Create Config
      </Button>
    </div>
  );
};

export default Sidebar;
