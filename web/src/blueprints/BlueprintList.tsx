import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import { toast } from "react-toastify";
import { Pencil, Trash2, Copy, Plus, Filter, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";

// Define the Blueprint interface
interface Blueprint {
  id: string;
  name: string;
  description: string;
  type: string;
  version: string;
  created_by: number;
  is_public: boolean;
  created_at: string;
}

const BlueprintList = () => {
  const [blueprints, setBlueprints] = useState<Blueprint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [blueprintTypes, setBlueprintTypes] = useState<string[]>([]);

  const navigate = useNavigate();

  useEffect(() => {
    fetchBlueprints();
    fetchBlueprintTypes();
  }, []);

  // Fetch blueprints from API
  const fetchBlueprints = async () => {
    try {
      setIsLoading(true);
      const response = await axios.get("/api/blueprints");
      setBlueprints(response.data || []);
    } catch (error) {
      console.error("Failed to fetch blueprints:", error);
      toast.error("Failed to load blueprints");
    } finally {
      setIsLoading(false);
    }
  };

  // Fetch blueprint types from API
  const fetchBlueprintTypes = async () => {
    try {
      const response = await axios.get("/api/blueprint-types");
      setBlueprintTypes(response.data);
    } catch (error) {
      console.error("Failed to fetch blueprint types:", error);
    }
  };

  // Filter blueprints based on search query and type filter
  console.log("blueprints", blueprints);
  const filteredBlueprints = blueprints.filter((blueprint) => {
    const matchesSearch =
      blueprint.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      blueprint.description.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesType = typeFilter === "all" || blueprint.type === typeFilter;

    return matchesSearch && matchesType;
  });

  // Handle blueprint deletion
  const handleDeleteBlueprint = async (id: string) => {
    if (window.confirm("Are you sure you want to delete this blueprint?")) {
      try {
        await axios.delete(`/api/blueprints/${id}`);
        toast.success("Blueprint deleted successfully");
        fetchBlueprints(); // Refresh the list
      } catch (error) {
        console.error("Failed to delete blueprint:", error);
        toast.error("Failed to delete blueprint");
      }
    }
  };

  // Handle blueprint creation
  const handleCreateBlueprint = () => {
    navigate("/blueprints/new");
  };

  // Handle blueprint editing
  const handleEditBlueprint = (id: string) => {
    navigate(`/blueprints/edit/${id}`);
  };

  // Handle using a blueprint to create a config
  const handleUseBlueprint = (id: string) => {
    navigate(`/blueprints/use/${id}`);
  };

  // Get appropriate badge color based on blueprint type
  const getBadgeVariant = (type: string) => {
    switch (type) {
      case "nodejs":
        return "success";
      case "php":
        return "secondary";
      case "python":
        return "warning";
      case "static":
        return "default";
      default:
        return "outline";
    }
  };

  // Format date string
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  return (
    <div className="container mx-auto py-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold mb-2">Blueprint Templates</h1>
          <p className="text-gray-600">
            Create and manage reusable deployment templates
          </p>
        </div>
        <Button onClick={handleCreateBlueprint}>
          <Plus className="mr-2 h-4 w-4" /> Create Blueprint
        </Button>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col sm:flex-row gap-4 mb-6">
        <div className="relative flex-grow">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            className="pl-10"
            placeholder="Search blueprints..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2 w-full sm:w-auto">
          <Filter className="h-4 w-4 text-gray-400" />
          <Label>Type:</Label>
          <Select value={typeFilter} onValueChange={setTypeFilter}>
            <SelectTrigger className="w-full sm:w-40">
              <SelectValue placeholder="Filter by type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              {blueprintTypes.map((type) => (
                <SelectItem key={type} value={type}>
                  {type.charAt(0).toUpperCase() + type.slice(1)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {isLoading ? (
        <div className="flex justify-center p-10">
          <p>Loading blueprints...</p>
        </div>
      ) : filteredBlueprints.length === 0 ? (
        <div className="text-center p-10 bg-gray-50 rounded-lg">
          <p className="text-lg text-gray-600 mb-4">No blueprints found</p>
          <Button onClick={handleCreateBlueprint}>
            Create Your First Blueprint
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredBlueprints.map((blueprint) => (
            <Card key={blueprint.id} className="flex flex-col">
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div>
                    <Badge variant={getBadgeVariant(blueprint.type)}>
                      {blueprint.type}
                    </Badge>
                    {blueprint.is_public && (
                      <Badge variant="outline" className="ml-2">
                        Public
                      </Badge>
                    )}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm">
                        ⋮
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        onClick={() => handleUseBlueprint(blueprint.id)}
                      >
                        <Copy className="mr-2 h-4 w-4" />
                        Use Template
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onClick={() => handleEditBlueprint(blueprint.id)}
                      >
                        <Pencil className="mr-2 h-4 w-4" />
                        Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-red-600"
                        onClick={() => handleDeleteBlueprint(blueprint.id)}
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <CardTitle>{blueprint.name}</CardTitle>
                <CardDescription>v{blueprint.version}</CardDescription>
              </CardHeader>
              <CardContent className="flex-grow">
                <p className="text-gray-600 line-clamp-3">
                  {blueprint.description}
                </p>
              </CardContent>
              <CardFooter className="border-t pt-3 flex justify-between items-center">
                <span className="text-xs text-gray-500">
                  Created {formatDate(blueprint.created_at)}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleUseBlueprint(blueprint.id)}
                >
                  Use Template
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default BlueprintList;
