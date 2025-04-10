import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import axios from "axios";
import { toast } from "react-toastify";
import { ArrowLeft, ArrowRight, Check, Eye } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Task {
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

const BlueprintUse = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [blueprint, setBlueprint] = useState<Blueprint | null>(null);
  const [configName, setConfigName] = useState("");
  const [parameters, setParameters] = useState<Record<string, string>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState(false);
  const [step, setStep] = useState(1);
  const [showPreview, setShowPreview] = useState(false);
  const [generatedConfig, setGeneratedConfig] = useState<any>(null);

  // Fetch blueprint data
  useEffect(() => {
    const fetchBlueprint = async () => {
      try {
        setIsLoading(true);
        const response = await axios.get(`/api/blueprints/${id}`);
        setBlueprint(response.data.data);

        // Initialize parameters with blueprint defaults
        setParameters(response.data.parameters || {});
      } catch (error) {
        console.error("Failed to fetch blueprint:", error);
        toast.error("Failed to load blueprint");
        navigate("/blueprints");
      } finally {
        setIsLoading(false);
      }
    };
    if (id) fetchBlueprint();
  }, [id, navigate]);

  // Handle parameter change
  const handleParameterChange = (key: string, value: string) => {
    setParameters((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  // Generate config from blueprint
  const handleGenerateConfig = async () => {
    if (!configName.trim()) {
      toast.error("Configuration name is required");
      return;
    }

    try {
      setIsGenerating(true);
      const response = await axios.post("/api/blueprints/generate", {
        blueprint_id: id,
        config_name: configName,
        parameters: parameters,
      });

      setGeneratedConfig(response.data);
      setStep(3);
      toast.success("Configuration generated successfully");
    } catch (error) {
      console.error("Failed to generate config:", error);
      toast.error("Failed to generate configuration");
    } finally {
      setIsGenerating(false);
    }
  };

  // Toggle JSON preview dialog
  const togglePreview = () => {
    setShowPreview(!showPreview);
  };

  // Navigate steps
  const nextStep = () => {
    if (step < 3) setStep(step + 1);
  };

  const prevStep = () => {
    if (step > 1) setStep(step - 1);
  };

  // Save generated config to configs page
  const handleSaveConfig = async () => {
    try {
      await axios.post("/api/configs", generatedConfig);
      toast.success("Configuration saved successfully");
      navigate("/configs");
    } catch (error) {
      console.error("Failed to save config:", error);
      toast.error("Failed to save configuration");
    }
  };

  // Render loading state
  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <p>Loading blueprint...</p>
      </div>
    );
  }

  // Render error state
  if (!blueprint) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <p>Error loading blueprint</p>
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
          <h1 className="text-2xl font-bold">Use Blueprint</h1>
          <p className="text-gray-600">
            Generate a configuration from the {blueprint.name} blueprint
          </p>
        </div>
      </div>

      <div className="max-w-2xl mx-auto">
        <Progress
          value={step === 1 ? 33 : step === 2 ? 66 : 100}
          className="mb-6"
        />

        {/* Step 1: Blueprint Overview */}
        {step === 1 && (
          <Card>
            <CardHeader>
              <CardTitle>Blueprint Overview</CardTitle>
              <CardDescription>
                Review the blueprint details before generating a configuration
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <div>
                  <h2 className="text-xl font-semibold">{blueprint.name}</h2>
                  <Badge variant="secondary" className="mt-2">
                    {blueprint.type} v{blueprint.version}
                  </Badge>
                </div>
                {blueprint.is_public && (
                  <Badge variant="outline">Public Blueprint</Badge>
                )}
              </div>
              <p className="text-gray-600">{blueprint.description}</p>

              <div className="bg-gray-50 p-4 rounded-md">
                <h3 className="font-medium mb-2">Tasks</h3>
                {blueprint.tasks.map((task, index) => (
                  <div key={index} className="mb-2">
                    <div className="flex justify-between">
                      <span className="font-semibold">{task.name}</span>
                      {task.depends_on && (
                        <span className="text-sm text-gray-500">
                          Depends on: {task.depends_on.join(", ")}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
            <CardFooter>
              <Button className="ml-auto" onClick={nextStep}>
                Next: Configure Parameters
                <ArrowRight className="h-4 w-4 ml-2" />
              </Button>
            </CardFooter>
          </Card>
        )}

        {/* Step 2: Configure Parameters */}
        {step === 2 && (
          <Card>
            <CardHeader>
              <CardTitle>Configure Parameters</CardTitle>
              <CardDescription>
                Customize the parameters for your configuration
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Configuration Name</Label>
                <Input
                  value={configName}
                  onChange={(e) => setConfigName(e.target.value)}
                  placeholder="Enter a name for your configuration"
                  required
                />
              </div>

              {Object.entries(parameters).map(([key, defaultValue]) => (
                <div key={key} className="space-y-2">
                  <Label>{key}</Label>
                  <Input
                    value={parameters[key]}
                    onChange={(e) => handleParameterChange(key, e.target.value)}
                    placeholder={`Enter value for ${key} (default: ${defaultValue})`}
                  />
                  {defaultValue && (
                    <p className="text-sm text-gray-500">
                      Default value: {defaultValue}
                    </p>
                  )}
                </div>
              ))}
            </CardContent>
            <CardFooter className="flex justify-between">
              <Button variant="outline" onClick={prevStep}>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Previous
              </Button>
              <Button
                onClick={handleGenerateConfig}
                disabled={isGenerating || !configName.trim()}
              >
                {isGenerating ? "Generating..." : "Generate Configuration"}
                <ArrowRight className="h-4 w-4 ml-2" />
              </Button>
            </CardFooter>
          </Card>
        )}

        {/* Step 3: Generated Configuration */}
        {step === 3 && generatedConfig && (
          <Card>
            <CardHeader>
              <CardTitle>Generated Configuration</CardTitle>
              <CardDescription>
                Review the generated configuration before saving
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="bg-gray-50 p-4 rounded-md relative">
                <Button
                  variant="ghost"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={togglePreview}
                >
                  <Eye className="h-4 w-4 mr-2" /> Full Preview
                </Button>
                <pre className="text-sm overflow-x-auto max-h-80 overflow-y-auto">
                  {JSON.stringify(generatedConfig, null, 2)}
                </pre>
              </div>
            </CardContent>
            <CardFooter className="flex justify-between">
              <Button variant="outline" onClick={prevStep}>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
              <Button onClick={handleSaveConfig}>
                <Check className="h-4 w-4 mr-2" />
                Save Configuration
              </Button>
            </CardFooter>
          </Card>
        )}

        <Dialog open={showPreview} onOpenChange={setShowPreview}>
          <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Full Configuration Preview</DialogTitle>
              <DialogDescription>
                Detailed view of the generated configuration
              </DialogDescription>
            </DialogHeader>
            <div className="p-4">
              <pre className="text-sm bg-gray-50 p-4 rounded-md overflow-x-auto">
                {JSON.stringify(generatedConfig, null, 2)}
              </pre>
            </div>
            <div className="flex justify-end p-4">
              <DialogClose asChild>
                <Button variant="outline">Close</Button>
              </DialogClose>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
};

export default BlueprintUse;
