import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import { v4 as uuidv4 } from "uuid";
import { toast, ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./components/ui/table";
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from "./components/ui/select";

import { Input } from "./components/ui/input";
import { Button } from "./components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "./components/ui/dropdown-menu";
import { Avatar, AvatarFallback, AvatarImage } from "./components/ui/avatar";
import { Badge } from "./components/ui/badge";
import { Checkbox } from "./components/ui/checkbox";
import { MoreHorizontal, LogOut, Search, User } from "lucide-react";
import { useAuth } from "./contexts/AuthContext";
import Sidebar from "./components/sidebar/Sidebar";

export interface ConfigEntry {
  name: string;
  description: string;
  path: string;
  status?: "DEPLOYED" | "DRAFT" | "TEMPLATE"; // Add status field
}

export interface ConfigState {
  selectedHost: string;
  selectedTask: string;
  tasks: string[];
  hosts: string[];
}

interface ConfigDetails {
  tasks: string[];
  hosts: string[];
}

const App: React.FC = () => {
  const [configs, setConfigs] = useState<ConfigEntry[]>([]);
  const [newConfigPath, setNewConfigPath] = useState("");
  const [logs, setLogs] = useState<string[]>([]);
  const [configStates, setConfigStates] = useState<{
    [key: string]: ConfigState;
  }>({});

  const { logout } = useAuth();
  const navigate = useNavigate();

  const fetchConfigs = async () => {
    try {
      const response = await axios.get("/api/configs");

      console.log("Fetched Configs", response.data);

      const fetchedConfigs = Array.isArray(response.data) ? response.data : [];
      // Add a default status to each config for demo purposes
      const configsWithStatus = fetchedConfigs.map(
        (config: ConfigEntry, index: number) => ({
          ...config,
          status: ["DEPLOYED", "DRAFT", "TEMPLATE"][index % 3] as
            | "DEPLOYED"
            | "DRAFT"
            | "TEMPLATE",
        }),
      );
      setConfigs(configsWithStatus);
      const initialStates: { [key: string]: ConfigState } = {};
      configsWithStatus.forEach((config: ConfigEntry) => {
        initialStates[config.path] = {
          selectedHost: "all",
          selectedTask: "none",
          tasks: [],
          hosts: [],
        };
      });
      setConfigStates(initialStates);
    } catch (error) {
      console.error("Failed to fetch configs:", error);
      setConfigs([]);
      toast.error("Failed to fetch configurations. Please try again later.");
    }
  };

  // Fetch tasks and hosts for a config
  const fetchTasksAndHosts = async (configPath: string) => {
    try {
      const response = await axios.get(
        `/api/config-details?path=${encodeURIComponent(configPath)}`,
      );
      const { tasks, hosts }: ConfigDetails = response.data;
      setConfigStates((prev) => ({
        ...prev,
        [configPath]: {
          ...prev[configPath],
          tasks,
          hosts,
        },
      }));
    } catch (error) {
      console.error("Failed to fetch tasks and hosts:", error);
      setLogs((prevLogs) => [
        ...prevLogs,
        `Error fetching tasks and hosts for ${configPath}: ${error}`,
      ]);
    }
  };

  const updateConfig = (
    index: number,
    field: keyof ConfigEntry,
    value: string,
  ) => {
    setConfigs((prev) => {
      const newConfigs = [...prev];
      newConfigs[index] = { ...newConfigs[index], [field]: value };
      return newConfigs;
    });
  };

  const saveConfig = async (index: number) => {
    const config = configs[index];
    setLogs((prevLogs) => [...prevLogs, `Saving config: ${config.path}`]);
    try {
      await axios.post("/api/configs", config);
      toast.success("Config saved successfully!");
      fetchConfigs();
    } catch (error) {
      console.error("Failed to save config:", error);
      setLogs((prevLogs) => [
        ...prevLogs,
        `Error saving config ${config.path}: ${error}`,
      ]);
      toast.error("Failed to save config.");
    }
  };

  const addConfig = () => {
    if (newConfigPath) {
      const newConfig: ConfigEntry = {
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

  const deployConfig = async (configPath: string) => {
    setLogs((prevLogs) => [...prevLogs, `Deploying config: ${configPath}`]);
    const sessionID = uuidv4();
    setLogs([]); // Clear previous logs

    const ws = new WebSocket(`ws://localhost:8080/ws/logs/${sessionID}`);
    ws.onmessage = (event) => {
      setLogs((prevLogs) => [...prevLogs, event.data]);
    };
    ws.onclose = () => {
      setLogs((prevLogs) => [...prevLogs, "--- Deployment finished ---"]);
    };

    try {
      await axios.post("/api/deploy", {
        configPath,
        host: configStates[configPath].selectedHost,
        sessionID,
      });
      // Update status to DEPLOYED after successful deployment
      setConfigs((prev) =>
        prev.map((config) =>
          config.path === configPath
            ? { ...config, status: "DEPLOYED" }
            : config,
        ),
      );
      toast.success("Config deployed successfully!");
    } catch (error) {
      console.error("Failed to deploy config:", error);
      setLogs((prevLogs) => [...prevLogs, `Error: ${error}`]);
      toast.error("Failed to deploy config.");
    }
  };

  const executeTask = async (configPath: string) => {
    const selectedTask = configStates[configPath].selectedTask;
    if (selectedTask === "none") {
      setLogs((prevLogs) => [...prevLogs, "Error: Please select a task"]);
      toast.error("Please select a task to execute.");
      return;
    }

    const sessionID = uuidv4();
    setLogs([]); // Clear previous logs

    const ws = new WebSocket(`ws://localhost:8080/ws/logs/${sessionID}`);
    ws.onmessage = (event) => {
      setLogs((prevLogs) => [...prevLogs, event.data]);
    };
    ws.onclose = () => {
      setLogs((prevLogs) => [...prevLogs, "--- Task execution finished ---"]);
    };

    try {
      await axios.post("/api/task", {
        configPath,
        host: configStates[configPath].selectedHost,
        taskName: selectedTask,
        sessionID,
      });
      toast.success("Task executed successfully!");
    } catch (error) {
      console.error("Failed to execute task:", error);
      setLogs((prevLogs) => [...prevLogs, `Error: ${error}`]);
      toast.error("Failed to execute task.");
    }
  };
  // Update selected host or task for a config
  const updateConfigState = (
    configPath: string,
    field: "selectedHost" | "selectedTask",
    value: string,
  ) => {
    setConfigStates((prev) => ({
      ...prev,
      [configPath]: {
        ...prev[configPath],
        [field]: value,
      },
    }));
  };

  const removeConfig = (index: number) => {
    setConfigs((prev) => {
      const updatedConfigs = [...prev];
      updatedConfigs.splice(index, 1);
      return updatedConfigs;
    });
    toast.success("Config removed successfully!");
  };

  const handleLogout = () => {
    logout();
    toast.success("Logged out successfully!");
    navigate("/login");
  };

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    addConfig();
  };

  useEffect(() => {
    fetchConfigs();
  }, []);

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Header */}
      <header className="absolute top-0 left-0 right-0 bg-primary-500 text-white p-4 flex justify-between items-center z-10">
        <h1 className="text-2xl font-bold">NyatiCtl</h1>
        <div className="flex items-center space-x-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
            <Input className="pl-10 bg-white text-black" placeholder="Search" />
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <div className="flex items-center space-x-2 cursor-pointer">
                <Avatar>
                  <AvatarImage src="https://github.com/shadcn.png" alt="User" />
                  <AvatarFallback>JD</AvatarFallback>
                </Avatar>
                <span>John Doe</span>
              </div>
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

      {/* Sidebar */}
      <Sidebar />

      {/* Main Content */}
      <div className="flex-1 flex flex-col pt-16">
        <main className="flex-1 p-6 overflow-auto">
          <div className="mb-4">
            <h2 className="text-2xl font-semibold">Manage Configs</h2>
            <p className="text-gray-600">
              Manage your configurations from the same page.
            </p>
          </div>
          <div className="mb-4">
            <form onSubmit={handleFormSubmit}>
              <Input
                placeholder="Config Path (e.g., nyati.live.yml)"
                value={newConfigPath}
                onChange={(e) => setNewConfigPath(e.target.value)}
                className="max-w-md"
              />
            </form>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <Checkbox />
                </TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Owner</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Parent</TableHead>
                <TableHead>Hosts</TableHead>
                <TableHead>Tasks</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {configs.map((config, index) => (
                <TableRow key={config.path}>
                  <TableCell>
                    <Checkbox />
                  </TableCell>
                  <TableCell>
                    <Input
                      value={config.name}
                      onChange={(e) =>
                        updateConfig(index, "name", e.target.value)
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <Avatar>
                        <AvatarImage
                          src={`https://i.pravatar.cc/150?img=${index + 1}`}
                          alt="Owner"
                        />
                        <AvatarFallback>
                          {config.name ? config.name[0] : "U"}
                        </AvatarFallback>
                      </Avatar>
                      <span>{config.name || "Unknown"}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        config.status === "DEPLOYED"
                          ? "success"
                          : config.status === "DRAFT"
                            ? "secondary"
                            : "warning"
                      }
                    >
                      {config.status}
                    </Badge>
                  </TableCell>
                  <TableCell>None</TableCell>

                  <TableCell>
                    <div className="min-w-[120px]">
                      <Select
                        value={configStates[config.path]?.selectedHost || "all"}
                        onValueChange={(value) =>
                          updateConfigState(config.path, "selectedHost", value)
                        }
                        onOpenChange={(open) => {
                          if (open) fetchTasksAndHosts(config.path);
                        }}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Select host" />
                        </SelectTrigger>
                        <SelectContent>
                          {configStates[config.path]?.hosts.map((host) => (
                            <SelectItem key={host} value={host}>
                              {host}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </TableCell>

                  <TableCell>
                    <div className="min-w-[120px]">
                      <Select
                        value={
                          configStates[config.path]?.selectedTask || "none"
                        }
                        onValueChange={(value) =>
                          updateConfigState(config.path, "selectedTask", value)
                        }
                        onOpenChange={(open) => {
                          if (open) fetchTasksAndHosts(config.path);
                        }}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Select task" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">None</SelectItem>
                          {configStates[config.path]?.tasks.map((task) => (
                            <SelectItem key={task} value={task}>
                              {task}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </TableCell>

                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => saveConfig(index)}>
                          Save
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => deployConfig(config.path)}
                        >
                          Deploy
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => executeTask(config.path)}
                        >
                          Execute Task
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem>Edit</DropdownMenuItem>
                        <DropdownMenuItem>Share</DropdownMenuItem>
                        <DropdownMenuItem>Copy Link</DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-red-600"
                          onClick={() => removeConfig(index)}
                        >
                          Remove
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-4 flex justify-between items-center">
            <p className="text-sm text-gray-600">
              Showing 1 to {configs.length} of {configs.length} entries
            </p>
            <div className="flex space-x-2">
              <Button variant="outline" size="sm">
                1
              </Button>
              <Button variant="outline" size="sm" disabled>
                2
              </Button>
              <Button variant="outline" size="sm" disabled>
                3
              </Button>
            </div>
          </div>
          <div className="mt-4">
            <h2 className="text-xl font-semibold">Logs</h2>
            <pre className="bg-gray-200 p-2 rounded max-h-60 overflow-auto">
              {logs.map((log, index) => (
                <div key={index}>{log}</div>
              ))}
            </pre>
          </div>
        </main>
      </div>
      <ToastContainer />
    </div>
  );
};

export default App;
