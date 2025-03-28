import React, { useState, useEffect } from "react";
import axios from "axios";
import { v4 as uuidv4 } from "uuid";
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Box,
  Typography,
} from "@mui/material";

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

  // Fetch configs on mount
  useEffect(() => {
    fetchConfigs();
  }, []);

  // Fetch configs from the backend
  const fetchConfigs = async () => {
    try {
      const response = await axios.get("/api/configs");
      setConfigs(response.data);
      // Initialize state for each config
      const initialStates: { [key: string]: ConfigState } = {};
      response.data.forEach((config: ConfigEntry) => {
        initialStates[config.path] = {
          selectedHost: "all",
          selectedTask: "",
          tasks: [],
          hosts: [],
        };
      });
      setConfigStates(initialStates);
    } catch (error) {
      console.error("Failed to fetch configs:", error);
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

  // Save a config
  const saveConfig = async (index: number) => {
    try {
      await axios.post("/api/configs", configs[index]);
      fetchConfigs();
    } catch (error) {
      console.error("Failed to save config:", error);
    }
  };

  // Add a new config
  const addConfig = () => {
    if (newConfigPath) {
      const newConfig: ConfigEntry = {
        name: "",
        description: "",
        path: newConfigPath,
      };
      setConfigs([...configs, newConfig]);
      setConfigStates((prev) => ({
        ...prev,
        [newConfig.path]: {
          selectedHost: "all",
          selectedTask: "",
          tasks: [],
          hosts: [],
        },
      }));
      setNewConfigPath("");
    }
  };

  // Update config field
  const updateConfig = (
    index: number,
    field: keyof ConfigEntry,
    value: string,
  ) => {
    const updatedConfigs = [...configs];
    updatedConfigs[index][field] = value;
    setConfigs(updatedConfigs);
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

  // Deploy a config
  const deployConfig = async (configPath: string) => {
    const sessionID = uuidv4();
    setLogs([]); // Clear previous logs

    // Connect to WebSocket for logs
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
      console.error("Failed to deploy:", error);
      setLogs((prevLogs) => [...prevLogs, `Error: ${error}`]);
    }
  };

  // Execute a specific task
  const executeTask = async (configPath: string) => {
    const selectedTask = configStates[configPath].selectedTask;
    if (!selectedTask) {
      setLogs((prevLogs) => [...prevLogs, "Error: Please select a task"]);
      return;
    }

    const sessionID = uuidv4();
    setLogs([]); // Clear previous logs

    // Connect to WebSocket for logs
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
    }
  };

  return (
    <Box sx={{ padding: 4 }}>
      <Typography variant="h4" gutterBottom>
        Nyatictl Web UI
      </Typography>

      {/* Add new config */}
      <Box sx={{ marginBottom: 4 }}>
        <TextField
          label="Config Path (e.g., nyati.live.yml)"
          value={newConfigPath}
          onChange={(e) => setNewConfigPath(e.target.value)}
          sx={{ marginRight: 2 }}
        />
        <Button variant="contained" onClick={addConfig}>
          Add Config
        </Button>
      </Box>

      {/* Configs table */}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Config Path</TableCell>
              <TableCell>Host</TableCell>
              <TableCell>Task</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {configs.map((config, index) => (
              <TableRow key={config.path}>
                <TableCell>
                  <TextField
                    value={config.name}
                    onChange={(e) =>
                      updateConfig(index, "name", e.target.value)
                    }
                  />
                </TableCell>
                <TableCell>
                  <TextField
                    value={config.description}
                    onChange={(e) =>
                      updateConfig(index, "description", e.target.value)
                    }
                  />
                </TableCell>
                <TableCell>{config.path}</TableCell>
                <TableCell>
                  <FormControl sx={{ minWidth: 120 }}>
                    <InputLabel>Host</InputLabel>
                    <Select
                      value={configStates[config.path]?.selectedHost || "all"}
                      onChange={(e) =>
                        updateConfigState(
                          config.path,
                          "selectedHost",
                          e.target.value,
                        )
                      }
                      onOpen={() => fetchTasksAndHosts(config.path)}
                    >
                      {configStates[config.path]?.hosts.map((host) => (
                        <MenuItem key={host} value={host}>
                          {host}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </TableCell>
                <TableCell>
                  <FormControl sx={{ minWidth: 120 }}>
                    <InputLabel>Task</InputLabel>
                    <Select
                      value={configStates[config.path]?.selectedTask || ""}
                      onChange={(e) =>
                        updateConfigState(
                          config.path,
                          "selectedTask",
                          e.target.value,
                        )
                      }
                      onOpen={() => fetchTasksAndHosts(config.path)}
                    >
                      <MenuItem value="">None</MenuItem>
                      {configStates[config.path]?.tasks.map((task) => (
                        <MenuItem key={task} value={task}>
                          {task}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </TableCell>
                <TableCell>
                  <Button
                    variant="contained"
                    color="primary"
                    onClick={() => saveConfig(index)}
                    sx={{ marginRight: 1 }}
                  >
                    Save
                  </Button>
                  <Button
                    variant="contained"
                    color="secondary"
                    onClick={() => deployConfig(config.path)}
                    sx={{ marginRight: 1 }}
                  >
                    Deploy
                  </Button>
                  <Button
                    variant="contained"
                    color="info"
                    onClick={() => executeTask(config.path)}
                  >
                    Execute Task
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Logs */}
      <Box sx={{ marginTop: 4 }}>
        <Typography variant="h6">Logs</Typography>
        <Paper sx={{ padding: 2, maxHeight: 300, overflow: "auto" }}>
          {logs.map((log, index) => (
            <Typography key={index} variant="body2">
              {log}
            </Typography>
          ))}
        </Paper>
      </Box>
    </Box>
  );
};

export default App;
