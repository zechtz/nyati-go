import { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import {
  Play,
  Server,
  CheckCircle,
  XCircle,
  Settings2,
  AlertCircle,
} from "lucide-react";
import axios from "axios";

// Define types for our data structures
interface Config {
  id: number | string;
  name: string;
  description: string;
  path: string;
  status?: "DEPLOYED" | "DRAFT" | "TEMPLATE";
  user_id?: number;
}

interface ConfigDetails {
  tasks: string[];
  hosts: string[];
}

interface SimulationResult {
  name: string;
  host: string;
  successful: boolean;
  output: string;
}

const SandboxSimulator = () => {
  const [configs, setConfigs] = useState<Config[]>([]);
  const [selectedConfig, setSelectedConfig] = useState<string>("");
  const [configDetails, setConfigDetails] = useState<ConfigDetails | null>(
    null,
  );
  const [selectedHost, setSelectedHost] = useState<string>("all");
  const [simulationResults, setSimulationResults] = useState<
    SimulationResult[]
  >([]);
  const [simulating, setSimulating] = useState<boolean>(false);
  const [simulationProgress, setSimulationProgress] = useState<number>(0);
  const [error, setError] = useState<string>("");

  // Fetch all available configs
  useEffect(() => {
    const fetchConfigs = async () => {
      try {
        const response = await axios.get("/api/configs");
        setConfigs(Array.isArray(response.data) ? response.data : []);
      } catch (err) {
        setError("Failed to load configurations");
        console.error(err);
      }
    };
    fetchConfigs();
  }, []);

  // Fetch config details when a config is selected
  useEffect(() => {
    if (selectedConfig) {
      const fetchConfigDetails = async () => {
        try {
          const response = await axios.get(
            `/api/config-details?path=${encodeURIComponent(selectedConfig)}`,
          );
          const { hosts }: ConfigDetails = response.data;
          setConfigDetails(response.data);
          if (hosts.length > 0) {
            setSelectedHost(hosts[0]);
          }
        } catch (err) {
          setError("Failed to load config details");
          console.error(err);
        }
      };
      fetchConfigDetails();
    }
  }, [selectedConfig]);

  const handleSimulate = async () => {
    if (!selectedConfig) {
      setError("Please select a configuration first");
      return;
    }

    setError("");
    setSimulating(true);
    setSimulationProgress(0);
    setSimulationResults([]);

    // Mock simulation data - in a real app, this would come from a special endpoint
    // that validates and simulates the deployment without executing actual commands
    const simulatedTasks = configDetails?.tasks || [];
    const progressIncrement = 100 / (simulatedTasks.length || 1);

    // Simulate task execution with a delay to mimic real deployment
    let currentProgress = 0;
    const results: SimulationResult[] = [];

    for (const task of simulatedTasks) {
      // Add a small delay to simulate processing
      await new Promise((resolve) => setTimeout(resolve, 800));

      // Simulate a success rate of about 90%
      const isSuccessful = Math.random() > 0.1;

      results.push({
        name: task,
        host: selectedHost,
        successful: isSuccessful,
        output: isSuccessful
          ? `[SANDBOX] Successfully executed ${task} on ${selectedHost}`
          : `[SANDBOX] Failed to execute ${task} on ${selectedHost}: Connection timeout`,
      });

      currentProgress += progressIncrement;
      setSimulationProgress(Math.min(currentProgress, 99));
    }

    // Finalize the simulation
    setTimeout(() => {
      setSimulationProgress(100);
      setSimulationResults(results);
      setSimulating(false);
    }, 1000);
  };

  const getSuccessRate = () => {
    if (simulationResults.length === 0) return 0;
    const successfulTasks = simulationResults.filter(
      (result) => result.successful,
    ).length;
    return (successfulTasks / simulationResults.length) * 100;
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings2 className="h-5 w-5" />
            Deployment Sandbox
          </CardTitle>
          <CardDescription>
            Test your deployment configurations in a safe environment without
            affecting production servers
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Select Configuration</label>
            <Select
              value={selectedConfig}
              onValueChange={setSelectedConfig}
              disabled={simulating}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select a configuration" />
              </SelectTrigger>
              <SelectContent>
                {configs.map((config) => (
                  // Make sure config.path is not an empty string
                  <SelectItem
                    key={config.id || config.path}
                    value={config.path || `config-${config.id}`} // Provide a fallback value
                  >
                    {config.name || config.path || `Config ${config.id}`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {configDetails && (
            <div className="space-y-2">
              <label className="text-sm font-medium">Select Host</label>
              <Select
                value={selectedHost}
                onValueChange={setSelectedHost}
                disabled={simulating}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a host" />
                </SelectTrigger>
                <SelectContent>
                  {configDetails.hosts.map((host) => (
                    <SelectItem key={host} value={host}>
                      {host}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {configDetails && (
            <div className="space-y-2">
              <div className="flex justify-between">
                <label className="text-sm font-medium">Tasks to Simulate</label>
                <span className="text-xs text-gray-500">
                  {configDetails.tasks.length} tasks
                </span>
              </div>
              <div className="border rounded-md p-2 max-h-32 overflow-y-auto">
                <ul className="space-y-1">
                  {configDetails.tasks.map((task) => (
                    <li
                      key={task}
                      className="text-sm flex items-center p-1 hover:bg-gray-50 rounded"
                    >
                      <Play className="h-3 w-3 mr-2 text-gray-400" />
                      {task}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}

          {error && (
            <div className="bg-red-50 text-red-500 p-3 rounded-md flex items-center">
              <AlertCircle className="h-5 w-5 mr-2" />
              {error}
            </div>
          )}

          <Button
            className="w-full"
            disabled={!selectedConfig || simulating}
            onClick={handleSimulate}
          >
            <Play className="h-4 w-4 mr-2" />
            Start Sandbox Simulation
          </Button>

          {simulating && (
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm font-medium">
                  Simulation in progress...
                </span>
                <span className="text-sm">
                  {Math.round(simulationProgress)}%
                </span>
              </div>
              <Progress value={simulationProgress} />
            </div>
          )}
        </CardContent>
      </Card>

      {simulationResults.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              Simulation Results
            </CardTitle>
            <CardDescription>
              Summary of the simulated deployment tasks
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between items-center">
              <div>
                <div className="text-sm text-gray-500">Success Rate</div>
                <div className="text-2xl font-semibold">
                  {getSuccessRate().toFixed(0)}%
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-500">Tasks</div>
                <div className="text-2xl font-semibold">
                  {simulationResults.length}
                </div>
              </div>
              <div>
                <div className="text-sm text-gray-500">Host</div>
                <div className="text-lg font-semibold">{selectedHost}</div>
              </div>
            </div>

            <div className="border rounded-md overflow-hidden">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Task
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {simulationResults.map((result, index) => (
                    <tr
                      key={index}
                      className={
                        result.successful ? "bg-green-50" : "bg-red-50"
                      }
                    >
                      <td className="px-4 py-3 text-sm font-medium">
                        {result.name}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center">
                          {result.successful ? (
                            <CheckCircle className="h-5 w-5 text-green-500 mr-2" />
                          ) : (
                            <XCircle className="h-5 w-5 text-red-500 mr-2" />
                          )}
                          <span className="text-sm">
                            {result.successful ? "Success" : "Failed"}
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setSimulationResults([])}>
              Clear Results
            </Button>
            <Button onClick={handleSimulate} disabled={simulating}>
              Run Again
            </Button>
          </CardFooter>
        </Card>
      )}
    </div>
  );
};

export default SandboxSimulator;
