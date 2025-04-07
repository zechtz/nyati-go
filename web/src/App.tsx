import { useState, useEffect } from "react";
import { Link, useNavigate } from "react-router-dom";
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
import { Input } from "./components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./components/ui/select";
import { Button } from "./components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./components/ui/dropdown-menu";
import { Avatar, AvatarFallback, AvatarImage } from "./components/ui/avatar";
import {
  MoreHorizontal,
  LogOut,
  Plus,
  Search,
  Settings,
  User,
} from "lucide-react";

interface ConfigEntry {
  name: string;
  description: string;
  path: string;
}

interface ConfigDetails {
  tasks: string[];
  hosts: string[];
}

interface ConfigState {
  selectedHost: string;
  selectedTask: string;
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
  const [loadingStates, setLoadingStates] = useState<{
    [key: string]: boolean;
  }>({});
  const navigate = useNavigate();

  const fetchConfigs = async () => {
    try {
      const response = await axios.get("/api/configs");
      const fetchedConfigs = Array.isArray(response.data) ? response.data : [];
      setConfigs(fetchedConfigs);
      const initialStates: { [key: string]: ConfigState } = {};
      fetchedConfigs.forEach((config: ConfigEntry) => {
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

  const fetchTasksAndHosts = async (configPath: string) => {
    setLoadingStates((prev) => ({ ...prev, [configPath]: true }));
    try {
      const response = await axios.get(
        `/api/config-details?path=${encodeURIComponent(configPath)}`,
      );
      const { tasks, hosts }: ConfigDetails = response.data;
      const filteredTasks = tasks.filter((task) => task !== "");
      const filteredHosts = hosts.filter((host) => host !== "");
      setConfigStates((prev) => ({
        ...prev,
        [configPath]: {
          ...prev[configPath],
          tasks: filteredTasks,
          hosts: filteredHosts,
        },
      }));
    } catch (error) {
      console.error("Failed to fetch tasks and hosts:", error);
      setLogs((prevLogs) => [
        ...prevLogs,
        `Error fetching tasks and hosts for ${configPath}: ${error}`,
      ]);
      toast.error(`Failed to fetch tasks and hosts for ${configPath}.`);
    } finally {
      setLoadingStates((prev) => ({ ...prev, [configPath]: false }));
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

  const updateConfigState = (
    configPath: string,
    field: keyof ConfigState,
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
    } catch (error) {
      console.error("Failed to execute task:", error);
      setLogs((prevLogs) => [...prevLogs, `Error: ${error}`]);
      toast.error("Failed to execute task.");
    }
  };

  const handleLogout = () => {
    // Placeholder for logout logic (to be implemented after backend auth)
    toast.success("Logged out successfully!");
    navigate("/login");
  };

  useEffect(() => {
    fetchConfigs();
  }, []);

  return (
    <div className="flex h-screen bg-primary-500">
      {/* Sidebar */}
      <div className="w-64 bg-primary  text-white flex flex-col">
        <div className="p-4">
          <h1 className="text-2xl font-bold">NyatiCtl</h1>
        </div>
        <nav className="flex-1 p-4 space-y-2">
          <Link
            to="/"
            className="flex items-center p-2 rounded hover:bg-primary-100"
          >
            <span className="ml-2">Dashboard</span>
          </Link>
          <Link
            to="/configs"
            className="flex items-center p-2 rounded bg-primary-500/80"
          >
            <span className="ml-2">Manage Configs</span>
          </Link>
          <div className="mt-auto">
            <Link
              to="/settings"
              className="flex items-center p-2 rounded hover:bg-primary-500/80"
            >
              <Settings className="h-5 w-5" />
              <span className="ml-2">Settings</span>
            </Link>
          </div>
        </nav>
        <Button
          className="m-4 bg-secondary-500 hover:bg-secondary-300"
          onClick={addConfig}
        >
          <Plus className="h-5 w-5 mr-2" />
          Add Config
        </Button>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="bg-primary-500 text-white p-4 flex justify-between items-center">
          <h2 className="text-xl font-semibold">Manage Configs</h2>
          <div className="flex items-center space-x-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
              <Input
                className="pl-10 bg-white text-black"
                placeholder="Search"
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <div className="flex items-center space-x-2 cursor-pointer">
                  <Avatar>
                    <AvatarImage
                      src="https://github.com/shadcn.png"
                      alt="User"
                    />
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

        {/* Main Content */}
        <main className="flex-1 p-6 overflow-auto">
          <div className="mb-4">
            <p className="text-gray-600">
              Manage your configurations from the same page.
            </p>
          </div>
          <div className="mb-4">
            <Input
              placeholder="Config Path (e.g., nyati.live.yml)"
              value={newConfigPath}
              onChange={(e) => setNewConfigPath(e.target.value)}
              className="max-w-md"
            />
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Config Path</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Task</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {configs.map((config, index) => (
                <TableRow key={config.path}>
                  <TableCell>
                    <Input
                      value={config.name}
                      onChange={(e) =>
                        updateConfig(index, "name", e.target.value)
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      value={config.description}
                      onChange={(e) =>
                        updateConfig(index, "description", e.target.value)
                      }
                    />
                  </TableCell>
                  <TableCell>{config.path}</TableCell>
                  <TableCell>
                    <Select
                      value={configStates[config.path]?.selectedHost || "all"}
                      onValueChange={(value) =>
                        updateConfigState(config.path, "selectedHost", value)
                      }
                      onOpenChange={(isOpen) => {
                        if (isOpen) {
                          fetchTasksAndHosts(config.path);
                        }
                      }}
                    >
                      <SelectTrigger className="w-[120px]">
                        <SelectValue placeholder="Select host" />
                      </SelectTrigger>
                      <SelectContent>
                        {loadingStates[config.path] ? (
                          <SelectItem value="loading" disabled>
                            Loading...
                          </SelectItem>
                        ) : configStates[config.path]?.hosts?.length > 0 ? (
                          configStates[config.path].hosts.map((host) => (
                            <SelectItem key={host} value={host}>
                              {host}
                            </SelectItem>
                          ))
                        ) : (
                          <SelectItem value="all">All</SelectItem>
                        )}
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <Select
                      value={configStates[config.path]?.selectedTask || "none"}
                      onValueChange={(value) =>
                        updateConfigState(config.path, "selectedTask", value)
                      }
                      onOpenChange={(isOpen) => {
                        if (isOpen) {
                          fetchTasksAndHosts(config.path);
                        }
                      }}
                    >
                      <SelectTrigger className="w-[120px]">
                        <SelectValue placeholder="Select task" />
                      </SelectTrigger>
                      <SelectContent>
                        {loadingStates[config.path] ? (
                          <SelectItem value="loading" disabled>
                            Loading...
                          </SelectItem>
                        ) : (
                          <>
                            <SelectItem value="none">None</SelectItem>
                            {configStates[config.path]?.tasks?.length > 0 ? (
                              configStates[config.path].tasks.map((task) => (
                                <SelectItem key={task} value={task}>
                                  {task}
                                </SelectItem>
                              ))
                            ) : (
                              <SelectItem value="no-tasks" disabled>
                                No tasks available
                              </SelectItem>
                            )}
                          </>
                        )}
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Button onClick={() => saveConfig(index)}>Save</Button>
                      <Button
                        variant="secondary"
                        onClick={() => deployConfig(config.path)}
                      >
                        Deploy
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() => executeTask(config.path)}
                      >
                        Execute Task
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>Edit</DropdownMenuItem>
                          <DropdownMenuItem>Share</DropdownMenuItem>
                          <DropdownMenuItem>Copy Link</DropdownMenuItem>
                          <DropdownMenuItem className="text-red-600">
                            Remove
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
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
            <pre className="bg-gray-100 p-2 rounded max-h-60 overflow-auto">
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
