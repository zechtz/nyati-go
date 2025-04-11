import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import axios from "axios";
import { toast } from "react-toastify";
import { Plus, Minus, Save, ArrowLeft, Code } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Task {
  id: string;
  name: string;
  cmd: string;
  dir?: string;
  expect: number;
  message?: string;
  retry?: boolean;
  askpass?: boolean;
  lib?: boolean;
  output?: boolean;
  depends_on?: string[];
}

interface Blueprint {
  id: string;
  name: string;
  description: string;
  type: string;
  version: string;
  tasks: Task[];
  parameters: Record<string, string>;
  created_by?: number;
  is_public: boolean;
  created_at?: string;
}

const defaultTask: Task = {
  id: crypto.randomUUID(),
  name: "",
  cmd: "",
  dir: "",
  expect: 0,
  message: "",
  retry: false,
  askpass: false,
  lib: false,
  output: false,
  depends_on: [],
};

const defaultBlueprint: Blueprint = {
  id: "",
  name: "",
  description: "",
  type: "custom",
  version: "1.0.0",
  tasks: [{ ...defaultTask }],
  parameters: {},
  is_public: false,
};

const BlueprintForm = () => {
  const { id } = useParams<{ id: string }>();
  const isEditing = !!id;
  const navigate = useNavigate();

  const [blueprint, setBlueprint] = useState<Blueprint>(defaultBlueprint);
  const [blueprintTypes, setBlueprintTypes] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [showJsonEditor, setShowJsonEditor] = useState(false);
  const [jsonValue, setJsonValue] = useState("");
  const [newParamKey, setNewParamKey] = useState("");
  const [newParamValue, setNewParamValue] = useState("");

  // Fetch blueprint types
  useEffect(() => {
    const fetchBlueprintTypes = async () => {
      try {
        const response = await axios.get("/api/blueprint-types");
        setBlueprintTypes(response.data.data);
      } catch (error) {
        console.error("Failed to fetch blueprint types:", error);
        toast.error("Failed to load blueprint types");
      }
    };

    fetchBlueprintTypes();
  }, []);

  // Fetch blueprint data if editing
  useEffect(() => {
    const fetchBlueprint = async () => {
      if (!isEditing) return;

      try {
        setIsLoading(true);
        const response = await axios.get(`/api/blueprints/${id}`);
        const fetchedBlueprint = response.data.data;
        // Add ids to tasks if they don't exist
        fetchedBlueprint.tasks = fetchedBlueprint.tasks.map((task: Task) => ({
          ...task,
          id: task.id || crypto.randomUUID(),
        }));

        console.log("fetchedBlueprint", fetchedBlueprint);

        setBlueprint((prev) => ({
          ...prev,
          ...fetchedBlueprint,
        }));

        setJsonValue(JSON.stringify(fetchedBlueprint, null, 2));
      } catch (error) {
        console.error("Failed to fetch blueprint:", error);
        toast.error("Failed to load blueprint");
        navigate("/blueprints");
      } finally {
        setIsLoading(false);
      }
    };

    fetchBlueprint();
  }, [id, isEditing, navigate]);

  // Update JSON when blueprint changes
  useEffect(() => {
    setJsonValue(JSON.stringify(blueprint, null, 2));
  }, [blueprint]);

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const { name, value } = e.target;
    setBlueprint((prev) => ({ ...prev, [name]: value }));
  };

  const handleSelectChange = (name: keyof Blueprint, value: string) => {
    setBlueprint((prev) => ({ ...prev, [name]: value }));
  };

  const handleCheckboxChange = (name: keyof Blueprint, checked: boolean) => {
    setBlueprint((prev) => ({ ...prev, [name]: checked }));
  };

  const handleTaskChange = (index: number, field: keyof Task, value: any) => {
    setBlueprint((prev) => {
      const updatedTasks = [...prev.tasks];
      updatedTasks[index] = { ...updatedTasks[index], [field]: value };
      return { ...prev, tasks: updatedTasks };
    });
  };

  const handleAddTask = () => {
    setBlueprint((prev) => ({
      ...prev,
      tasks: [...prev.tasks, { ...defaultTask, id: crypto.randomUUID() }],
    }));
  };

  const handleRemoveTask = (index: number) => {
    setBlueprint((prev) => {
      const updatedTasks = [...prev.tasks];
      const removedTask = updatedTasks.splice(index, 1)[0];

      // Update dependencies for other tasks
      const removedTaskId = removedTask.id;
      updatedTasks.forEach((task) => {
        if (task.depends_on?.includes(removedTaskId)) {
          task.depends_on = task.depends_on.filter(
            (dep) => dep !== removedTaskId,
          );
        }
      });

      return { ...prev, tasks: updatedTasks };
    });
  };

  const handleAddParameter = () => {
    if (!newParamKey.trim()) {
      toast.error("Parameter key cannot be empty");
      return;
    }

    setBlueprint((prev) => ({
      ...prev,
      parameters: {
        ...prev.parameters,
        [newParamKey]: newParamValue,
      },
    }));

    setNewParamKey("");
    setNewParamValue("");
  };

  const handleRemoveParameter = (key: string) => {
    setBlueprint((prev) => {
      const updatedParams = { ...prev.parameters };
      delete updatedParams[key];
      return { ...prev, parameters: updatedParams };
    });
  };

  const handleJSONUpdate = () => {
    try {
      const updatedBlueprint = JSON.parse(jsonValue);
      setBlueprint((prev) => ({
        ...prev,
        ...updatedBlueprint,
      }));

      setShowJsonEditor(false);
      toast.success("Blueprint updated from JSON");
    } catch (error) {
      toast.error("Invalid JSON format");
    }
  };

  const handlePreviewFromType = async () => {
    try {
      const response = await axios.get(
        `/api/blueprints/preset/${blueprint.type}`,
      );

      // Keep the current name, description, and is_public settings
      const presetBlueprint = response.data.data;
      presetBlueprint.name = blueprint.name || presetBlueprint.name;
      presetBlueprint.description =
        blueprint.description || presetBlueprint.description;
      presetBlueprint.is_public = blueprint.is_public;

      setBlueprint((prev) => ({
        ...prev,
        ...presetBlueprint,
      }));

      toast.success(`Loaded ${blueprint.type} blueprint template`);
    } catch (error) {
      console.error("Failed to load blueprint preset:", error);
      toast.error("Failed to load blueprint preset");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validate blueprint
    if (!blueprint.name.trim()) {
      toast.error("Blueprint name is required");
      return;
    }

    if (blueprint.tasks.length === 0) {
      toast.error("At least one task is required");
      return;
    }

    // Validate tasks
    for (let i = 0; i < blueprint.tasks.length; i++) {
      const task = blueprint.tasks[i];
      if (!task.name.trim()) {
        toast.error(`Task #${i + 1} name is required`);
        return;
      }
      if (!task.cmd.trim()) {
        toast.error(`Task #${i + 1} command is required`);
        return;
      }
    }

    try {
      setIsSaving(true);
      await axios.post("/api/blueprints", blueprint);
      toast.success(
        `Blueprint ${isEditing ? "updated" : "created"} successfully`,
      );
      navigate("/blueprints");
    } catch (error) {
      console.error("Failed to save blueprint:", error);
      toast.error(`Failed to ${isEditing ? "update" : "create"} blueprint`);
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center p-10">
        <p>Loading blueprint...</p>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6">
      <div className="flex items-center mb-6">
        <Button
          variant="ghost"
          onClick={() => navigate("/blueprints")}
          className="mr-4"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back
        </Button>
        <div>
          <h1 className="text-2xl font-bold">
            {isEditing ? "Edit Blueprint" : "Create Blueprint"}
          </h1>
          <p className="text-gray-600">
            {isEditing
              ? "Update your deployment blueprint"
              : "Create a new reusable deployment template"}
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Form */}
          <div className="lg:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Blueprint Details</CardTitle>
                <CardDescription>
                  Define the basic information about your deployment template
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      name="name"
                      value={blueprint.name}
                      onChange={handleChange}
                      placeholder="Blueprint name"
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="type">Type</Label>
                    <div className="flex gap-2">
                      <Select
                        value={blueprint.type}
                        onValueChange={(value) =>
                          handleSelectChange("type", value)
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select a type" />
                        </SelectTrigger>
                        <SelectContent>
                          {blueprintTypes.map((type) => (
                            <SelectItem key={type} value={type}>
                              {type.charAt(0).toUpperCase() + type.slice(1)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={handlePreviewFromType}
                      >
                        Load Template
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="version">Version</Label>
                    <Input
                      id="version"
                      name="version"
                      value={blueprint.version}
                      onChange={handleChange}
                      placeholder="1.0.0"
                    />
                  </div>
                  <div className="space-y-2 flex items-end">
                    <div className="flex items-center space-x-2">
                      <Checkbox
                        id="is_public"
                        checked={blueprint.is_public}
                        onCheckedChange={(checked) =>
                          handleCheckboxChange("is_public", checked === true)
                        }
                      />
                      <Label htmlFor="is_public">Public Blueprint</Label>
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <textarea
                    id="description"
                    name="description"
                    value={blueprint.description}
                    onChange={handleChange}
                    placeholder="Describe this blueprint and its purpose"
                    className="w-full min-h-[100px] p-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </CardContent>
            </Card>

            {/* Tasks Section */}
            <Card>
              <CardHeader>
                <CardTitle>Tasks</CardTitle>
                <CardDescription>
                  Define the deployment tasks and their dependencies
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {blueprint.tasks.map((task, index) => (
                  <div key={task.id} className="border p-4 rounded-md relative">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute top-2 right-2 text-gray-500 hover:text-red-500"
                      onClick={() => handleRemoveTask(index)}
                    >
                      <Minus className="h-4 w-4" />
                    </Button>

                    <h3 className="text-lg font-medium mb-3">
                      Task #{index + 1}
                    </h3>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                      <div className="space-y-2">
                        <Label htmlFor={`task-${index}-name`}>Name</Label>
                        <Input
                          id={`task-${index}-name`}
                          value={task.name}
                          onChange={(e) =>
                            handleTaskChange(index, "name", e.target.value)
                          }
                          placeholder="Task name"
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor={`task-${index}-expect`}>
                          Expected Exit Code
                        </Label>
                        <Input
                          id={`task-${index}-expect`}
                          type="number"
                          value={task.expect}
                          onChange={(e) =>
                            handleTaskChange(
                              index,
                              "expect",
                              parseInt(e.target.value),
                            )
                          }
                          placeholder="0"
                        />
                      </div>
                    </div>

                    <div className="space-y-2 mb-4">
                      <Label htmlFor={`task-${index}-cmd`}>Command</Label>
                      <Input
                        id={`task-${index}-cmd`}
                        value={task.cmd}
                        onChange={(e) =>
                          handleTaskChange(index, "cmd", e.target.value)
                        }
                        placeholder="Shell command to execute"
                        required
                      />
                    </div>

                    <div className="space-y-2 mb-4">
                      <Label htmlFor={`task-${index}-dir`}>Directory</Label>
                      <Input
                        id={`task-${index}-dir`}
                        value={task.dir || ""}
                        onChange={(e) =>
                          handleTaskChange(index, "dir", e.target.value)
                        }
                        placeholder="Working directory (optional)"
                      />
                    </div>

                    <div className="space-y-2 mb-4">
                      <Label htmlFor={`task-${index}-message`}>
                        Success Message
                      </Label>
                      <Input
                        id={`task-${index}-message`}
                        value={task.message || ""}
                        onChange={(e) =>
                          handleTaskChange(index, "message", e.target.value)
                        }
                        placeholder="Message to display on success (optional)"
                      />
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                      <div className="space-y-2">
                        <Label htmlFor={`task-${index}-depends-on`}>
                          Depends On
                        </Label>
                        <Select
                          value=""
                          onValueChange={(value) => {
                            if (!value) return;

                            const dependsOn = task.depends_on || [];
                            if (!dependsOn.includes(value)) {
                              handleTaskChange(index, "depends_on", [
                                ...dependsOn,
                                value,
                              ]);
                            }
                          }}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select dependencies" />
                          </SelectTrigger>
                          <SelectContent>
                            {blueprint.tasks
                              .filter((_, i) => i !== index)
                              .map((t) => (
                                <SelectItem
                                  key={t.id}
                                  value={t.id}
                                  disabled={
                                    !t.name || task.depends_on?.includes(t.name)
                                  }
                                >
                                  {t.name || `Task #${index + 1}`}
                                </SelectItem>
                              ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        {task.depends_on && task.depends_on.length > 0 ? (
                          <div className="mt-6">
                            {task.depends_on.map((depId) => {
                              const depTask = blueprint.tasks.find(
                                (t) => t.id === depId,
                              );
                              return (
                                <span
                                  key={depId}
                                  className="inline-flex items-center bg-gray-100 text-gray-800 mr-2 px-2 py-1 rounded text-sm"
                                >
                                  {depTask?.name || "Unknown Task"}
                                  <button
                                    type="button"
                                    className="ml-1 text-gray-500 hover:text-red-500"
                                    onClick={() => {
                                      const updatedDeps =
                                        task.depends_on?.filter(
                                          (d) => d !== depId,
                                        ) || [];
                                      handleTaskChange(
                                        index,
                                        "depends_on",
                                        updatedDeps,
                                      );
                                    }}
                                  >
                                    ×
                                  </button>
                                </span>
                              );
                            })}
                          </div>
                        ) : (
                          <div className="mt-6 text-gray-500 text-sm">
                            No dependencies
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id={`task-${index}-retry`}
                          checked={task.retry || false}
                          onCheckedChange={(checked) =>
                            handleTaskChange(index, "retry", checked === true)
                          }
                        />
                        <Label htmlFor={`task-${index}-retry`}>
                          Retry on Failure
                        </Label>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id={`task-${index}-askpass`}
                          checked={task.askpass || false}
                          onCheckedChange={(checked) =>
                            handleTaskChange(index, "askpass", checked === true)
                          }
                        />
                        <Label htmlFor={`task-${index}-askpass`}>
                          Ask Password
                        </Label>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id={`task-${index}-lib`}
                          checked={task.lib || false}
                          onCheckedChange={(checked) =>
                            handleTaskChange(index, "lib", checked === true)
                          }
                        />
                        <Label htmlFor={`task-${index}-lib`}>
                          Library Task
                        </Label>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id={`task-${index}-output`}
                          checked={task.output || false}
                          onCheckedChange={(checked) =>
                            handleTaskChange(index, "output", checked === true)
                          }
                        />
                        <Label htmlFor={`task-${index}-output`}>
                          Show Output
                        </Label>
                      </div>
                    </div>
                  </div>
                ))}

                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={handleAddTask}
                >
                  <Plus className="h-4 w-4 mr-2" /> Add Task
                </Button>
              </CardContent>
            </Card>

            {/* Parameters Section */}
            <Card>
              <CardHeader>
                <CardTitle>Parameters</CardTitle>
                <CardDescription>
                  Define variables to be used in task commands
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                  <div className="space-y-2">
                    <Label htmlFor="param-key">Parameter Key</Label>
                    <Input
                      id="param-key"
                      value={newParamKey}
                      onChange={(e) => setNewParamKey(e.target.value)}
                      placeholder="e.g., branch"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="param-value">Parameter Value</Label>
                    <Input
                      id="param-value"
                      value={newParamValue}
                      onChange={(e) => setNewParamValue(e.target.value)}
                      placeholder="e.g., main"
                    />
                  </div>
                  <div className="flex items-end">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleAddParameter}
                      className="w-full"
                    >
                      <Plus className="h-4 w-4 mr-2" /> Add Parameter
                    </Button>
                  </div>
                </div>

                {Object.keys(blueprint.parameters).length > 0 ? (
                  <div className="border rounded-md p-4">
                    <h3 className="font-medium mb-2">Current Parameters</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {Object.entries(blueprint.parameters).map(
                        ([key, value]) => (
                          <div
                            key={key}
                            className="flex items-center justify-between p-2 bg-gray-50 rounded"
                          >
                            <div>
                              <span className="font-medium">${key}</span>
                              <span className="text-gray-500 ml-2">
                                {value}
                              </span>
                            </div>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleRemoveParameter(key)}
                              className="text-red-500"
                            >
                              <Minus className="h-4 w-4" />
                            </Button>
                          </div>
                        ),
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="text-gray-500 text-center py-4">
                    No parameters defined yet
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* JSON Editor Dialog */}
          <Dialog open={showJsonEditor} onOpenChange={setShowJsonEditor}>
            <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Edit Blueprint JSON</DialogTitle>
                <DialogDescription>
                  Edit the blueprint directly in JSON format
                </DialogDescription>
              </DialogHeader>
              <div className="py-4">
                <textarea
                  value={jsonValue}
                  onChange={(e) => setJsonValue(e.target.value)}
                  className="font-mono text-sm w-full h-[60vh] p-4 border rounded-md"
                />
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setShowJsonEditor(false)}
                >
                  Cancel
                </Button>
                <Button onClick={handleJSONUpdate}>Update Blueprint</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* Sidebar */}
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <Button type="submit" className="w-full" disabled={isSaving}>
                  <Save className="h-4 w-4 mr-2" />
                  {isSaving
                    ? isEditing
                      ? "Updating..."
                      : "Creating..."
                    : isEditing
                      ? "Update Blueprint"
                      : "Create Blueprint"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={() => setShowJsonEditor(true)}
                >
                  <Code className="h-4 w-4 mr-2" /> Edit as JSON
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Help</CardTitle>
              </CardHeader>
              <CardContent className="text-sm space-y-3">
                <p>
                  <strong>Tasks</strong> define the sequence of commands to run
                  during deployment.
                </p>
                <p>
                  <strong>Parameters</strong> are variables that can be used in
                  task commands using the <code>${"{parameter}"}</code> syntax.
                </p>
                <p>
                  <strong>Dependencies</strong> ensure tasks run in the correct
                  order.
                </p>
                <p className="text-gray-500 italic">
                  Tip: Load a template based on your application type to get
                  started.
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </form>
    </div>
  );
};

export default BlueprintForm;
