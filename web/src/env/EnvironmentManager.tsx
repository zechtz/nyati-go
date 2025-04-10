import React, { useState, useEffect } from "react";
import axios from "axios";
import { toast } from "react-toastify";
import {
  Plus,
  MoreHorizontal,
  Trash,
  Lock,
  Eye,
  EyeOff,
  Download,
  Upload,
  Check,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Checkbox } from "@/components/ui/checkbox";

// Define types
interface Environment {
  name: string;
  description: string;
  is_current: boolean;
  var_count: number;
  secret_count: number;
}

interface Variable {
  key: string;
  value: string;
  is_secret: boolean;
}

const EnvironmentManager: React.FC = () => {
  // State
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [currentEnv, setCurrentEnv] = useState<string>("");
  const [variables, setVariables] = useState<Variable[]>([]);
  const [encryptionKey, setEncryptionKey] = useState<string>("");
  const [showSecrets, setShowSecrets] = useState<boolean>(false);
  const [newEnvName, setNewEnvName] = useState<string>("");
  const [newEnvDescription, setNewEnvDescription] = useState<string>("");
  const [newVarKey, setNewVarKey] = useState<string>("");
  const [newVarValue, setNewVarValue] = useState<string>("");
  const [newVarIsSecret, setNewVarIsSecret] = useState<boolean>(false);
  const [isAddEnvDialogOpen, setIsAddEnvDialogOpen] = useState<boolean>(false);
  const [isAddVarDialogOpen, setIsAddVarDialogOpen] = useState<boolean>(false);
  const [, setIsLoading] = useState<boolean>(true);

  // Fetch environments on load
  useEffect(() => {
    fetchEnvironments();
  }, []);

  // Fetch environments when the current environment changes
  useEffect(() => {
    if (currentEnv) {
      fetchVariables(currentEnv);
    }
  }, [currentEnv]);

  // Fetch all environments
  const fetchEnvironments = async () => {
    setIsLoading(true);
    try {
      const response = await axios.get("/api/env/list");

      console.log("environments", response);

      setEnvironments(response.data);

      // Set current environment
      const current = response.data.find((env: Environment) => env.is_current);
      if (current) {
        setCurrentEnv(current.name);
      }
    } catch (error) {
      console.error("Failed to fetch environments:", error);
      toast.error("Failed to load environments");
    } finally {
      setIsLoading(false);
    }
  };

  // Fetch variables for an environment
  const fetchVariables = async (envName: string) => {
    setIsLoading(true);
    try {
      const headers: Record<string, string> = {};
      if (showSecrets && encryptionKey) {
        headers["X-Encryption-Key"] = encryptionKey;
      }

      const response = await axios.get(
        `/api/env/vars/${envName}?show_secrets=${showSecrets}`,
        { headers },
      );
      setVariables(response.data);
    } catch (error) {
      console.error(`Failed to fetch variables for ${envName}:`, error);
      toast.error("Failed to load variables");
    } finally {
      setIsLoading(false);
    }
  };

  // Create a new environment
  const createEnvironment = async () => {
    if (!newEnvName) {
      toast.error("Environment name is required");
      return;
    }

    try {
      await axios.post("/api/env/create", {
        name: newEnvName,
        description: newEnvDescription,
      });

      toast.success(`Environment "${newEnvName}" created successfully`);
      setIsAddEnvDialogOpen(false);
      setNewEnvName("");
      setNewEnvDescription("");
      fetchEnvironments();
    } catch (error) {
      console.error("Failed to create environment:", error);
      toast.error("Failed to create environment");
    }
  };

  // Switch to a different environment
  const switchEnvironment = async (envName: string) => {
    try {
      await axios.post(`/api/env/switch/${envName}`);
      setCurrentEnv(envName);

      // Update environment list to reflect current env
      setEnvironments((prevEnvs) =>
        prevEnvs.map((env) => ({
          ...env,
          is_current: env.name === envName,
        })),
      );

      toast.success(`Switched to environment "${envName}"`);
      fetchVariables(envName);
    } catch (error) {
      console.error(`Failed to switch to environment ${envName}:`, error);
      toast.error("Failed to switch environment");
    }
  };

  // Delete an environment
  const deleteEnvironment = async (envName: string) => {
    // Confirm before deleting
    if (
      !window.confirm(
        `Are you sure you want to delete environment "${envName}"?`,
      )
    ) {
      return;
    }

    try {
      await axios.delete(`/api/env/delete/${envName}`);
      toast.success(`Environment "${envName}" deleted successfully`);

      // If we deleted the current environment, reset
      if (currentEnv === envName) {
        const remainingEnvs = environments.filter(
          (env) => env.name !== envName,
        );
        if (remainingEnvs.length > 0) {
          setCurrentEnv(remainingEnvs[0].name);
          fetchVariables(remainingEnvs[0].name);
        } else {
          setCurrentEnv("");
          setVariables([]);
        }
      }

      fetchEnvironments();
    } catch (error) {
      console.error(`Failed to delete environment ${envName}:`, error);
      toast.error("Failed to delete environment");
    }
  };

  // Add a new variable
  const addVariable = async () => {
    if (!newVarKey) {
      toast.error("Variable key is required");
      return;
    }

    const headers: Record<string, string> = {};
    if (newVarIsSecret && encryptionKey) {
      headers["X-Encryption-Key"] = encryptionKey;
    } else if (newVarIsSecret && !encryptionKey) {
      toast.error("Encryption key is required for secrets");
      return;
    }

    try {
      await axios.post(
        `/api/env/vars/${currentEnv}`,
        {
          key: newVarKey,
          value: newVarValue,
          is_secret: newVarIsSecret,
        },
        { headers },
      );

      toast.success(`Variable "${newVarKey}" added successfully`);
      setIsAddVarDialogOpen(false);
      setNewVarKey("");
      setNewVarValue("");
      setNewVarIsSecret(false);
      fetchVariables(currentEnv);
    } catch (error) {
      console.error("Failed to add variable:", error);
      toast.error("Failed to add variable");
    }
  };

  // Delete a variable
  const deleteVariable = async (key: string) => {
    // Confirm before deleting
    if (!window.confirm(`Are you sure you want to delete variable "${key}"?`)) {
      return;
    }

    try {
      await axios.delete(`/api/env/vars/${currentEnv}/${key}`);
      toast.success(`Variable "${key}" deleted successfully`);
      fetchVariables(currentEnv);
    } catch (error) {
      console.error(`Failed to delete variable ${key}:`, error);
      toast.error("Failed to delete variable");
    }
  };

  // Export environment to .env file
  const exportEnvironment = async () => {
    const outputPath = prompt("Enter output path for .env file", ".env");
    if (!outputPath) return;

    const headers: Record<string, string> = {};
    if (encryptionKey) {
      headers["X-Encryption-Key"] = encryptionKey;
    }

    try {
      await axios.post(
        `/api/env/export/${currentEnv}`,
        {
          output_path: outputPath,
        },
        { headers },
      );

      toast.success(`Environment "${currentEnv}" exported to "${outputPath}"`);
    } catch (error) {
      console.error(`Failed to export environment ${currentEnv}:`, error);
      toast.error("Failed to export environment");
    }
  };

  // Import environment from .env file
  const importEnvironment = async () => {
    const inputPath = prompt("Enter input path for .env file", ".env");
    if (!inputPath) return;

    const asSecrets = window.confirm("Import as encrypted secrets?");

    const headers: Record<string, string> = {};
    if (asSecrets && encryptionKey) {
      headers["X-Encryption-Key"] = encryptionKey;
    } else if (asSecrets && !encryptionKey) {
      toast.error("Encryption key is required to import as secrets");
      return;
    }

    try {
      await axios.post(
        `/api/env/import/${currentEnv}`,
        {
          input_path: inputPath,
          as_secrets: asSecrets,
        },
        { headers },
      );

      toast.success(
        `Variables imported from "${inputPath}" to environment "${currentEnv}"`,
      );
      fetchVariables(currentEnv);
    } catch (error) {
      console.error(`Failed to import variables from ${inputPath}:`, error);
      toast.error("Failed to import variables");
    }
  };

  // Toggle showing secret values
  const toggleShowSecrets = () => {
    if (!showSecrets && !encryptionKey) {
      const key = prompt("Enter encryption key to view secrets");
      if (!key) return;
      setEncryptionKey(key);
    }

    setShowSecrets(!showSecrets);
    fetchVariables(currentEnv);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Environment Management</CardTitle>
          <CardDescription>
            Manage environments and variables for deployment
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex justify-between items-center mb-4">
            <div className="flex items-center space-x-2">
              <Label htmlFor="encryption-key">Encryption Key:</Label>
              <Input
                id="encryption-key"
                type="password"
                value={encryptionKey}
                onChange={(e) => setEncryptionKey(e.target.value)}
                placeholder="For encrypting/decrypting secrets"
                className="w-64"
              />
            </div>
            <Dialog
              open={isAddEnvDialogOpen}
              onOpenChange={setIsAddEnvDialogOpen}
            >
              <DialogTrigger asChild>
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Add Environment
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Create New Environment</DialogTitle>
                  <DialogDescription>
                    Add a new deployment environment
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="env-name">Environment Name</Label>
                    <Input
                      id="env-name"
                      value={newEnvName}
                      onChange={(e) => setNewEnvName(e.target.value)}
                      placeholder="e.g., production, staging"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="env-desc">Description</Label>
                    <Input
                      id="env-desc"
                      value={newEnvDescription}
                      onChange={(e) => setNewEnvDescription(e.target.value)}
                      placeholder="Description of this environment"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button onClick={createEnvironment}>
                    Create Environment
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          {/* Environments Table */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold mb-2">Environments</h3>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Variables</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {environments.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center">
                      No environments found
                    </TableCell>
                  </TableRow>
                ) : (
                  environments.map((env) => (
                    <TableRow key={env.name}>
                      <TableCell className="font-medium">{env.name}</TableCell>
                      <TableCell>{env.description}</TableCell>
                      <TableCell>
                        {env.is_current ? (
                          <Badge variant="success">Current</Badge>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => switchEnvironment(env.name)}
                          >
                            Use
                          </Button>
                        )}
                      </TableCell>
                      <TableCell>
                        {env.var_count} variables, {env.secret_count} secrets
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={() => switchEnvironment(env.name)}
                              disabled={env.is_current}
                            >
                              <Check className="h-4 w-4 mr-2" />
                              Use Environment
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => deleteEnvironment(env.name)}
                              disabled={env.is_current}
                              className="text-red-600"
                            >
                              <Trash className="h-4 w-4 mr-2" />
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Variables Section */}
          {currentEnv && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold">
                  Variables for {currentEnv}
                </h3>
                <div className="flex space-x-2">
                  <Button
                    variant="outline"
                    onClick={toggleShowSecrets}
                    className="flex items-center"
                  >
                    {showSecrets ? (
                      <>
                        <EyeOff className="h-4 w-4 mr-2" />
                        Hide Secrets
                      </>
                    ) : (
                      <>
                        <Eye className="h-4 w-4 mr-2" />
                        Show Secrets
                      </>
                    )}
                  </Button>
                  <Button
                    variant="outline"
                    onClick={exportEnvironment}
                    className="flex items-center"
                  >
                    <Download className="h-4 w-4 mr-2" />
                    Export
                  </Button>
                  <Button
                    variant="outline"
                    onClick={importEnvironment}
                    className="flex items-center"
                  >
                    <Upload className="h-4 w-4 mr-2" />
                    Import
                  </Button>
                  <Dialog
                    open={isAddVarDialogOpen}
                    onOpenChange={setIsAddVarDialogOpen}
                  >
                    <DialogTrigger asChild>
                      <Button>
                        <Plus className="h-4 w-4 mr-2" />
                        Add Variable
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Add Variable</DialogTitle>
                        <DialogDescription>
                          Add a new variable to the {currentEnv} environment
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4 py-4">
                        <div className="space-y-2">
                          <Label htmlFor="var-key">Variable Key</Label>
                          <Input
                            id="var-key"
                            value={newVarKey}
                            onChange={(e) => setNewVarKey(e.target.value)}
                            placeholder="e.g., DATABASE_URL"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="var-value">Value</Label>
                          <Input
                            id="var-value"
                            value={newVarValue}
                            onChange={(e) => setNewVarValue(e.target.value)}
                            placeholder="Variable value"
                          />
                        </div>
                        <div className="flex items-center space-x-2">
                          <Checkbox
                            id="var-secret"
                            checked={newVarIsSecret}
                            onCheckedChange={(checked) =>
                              setNewVarIsSecret(checked === true)
                            }
                          />
                          <Label htmlFor="var-secret">
                            Store as encrypted secret
                          </Label>
                        </div>
                      </div>
                      <DialogFooter>
                        <Button onClick={addVariable}>Add Variable</Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>
              </div>

              {/* Variables Table */}
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Key</TableHead>
                    <TableHead>Value</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {variables && variables.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center">
                        No variables found
                      </TableCell>
                    </TableRow>
                  ) : (
                    variables &&
                    variables.map((variable) => (
                      <TableRow key={variable.key}>
                        <TableCell className="font-medium">
                          {variable.key}
                        </TableCell>
                        <TableCell>
                          {variable.is_secret ? (
                            <div className="flex items-center">
                              {showSecrets ? (
                                variable.value
                              ) : (
                                <>
                                  <Lock className="h-4 w-4 mr-1" />
                                  <span className="text-gray-500">
                                    ●●●●●●●●
                                  </span>
                                </>
                              )}
                            </div>
                          ) : (
                            variable.value
                          )}
                        </TableCell>
                        <TableCell>
                          {variable.is_secret ? (
                            <Badge variant="warning">Secret</Badge>
                          ) : (
                            <Badge variant="secondary">Regular</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => deleteVariable(variable.key)}
                          >
                            <Trash className="h-4 w-4 text-red-500" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default EnvironmentManager;
