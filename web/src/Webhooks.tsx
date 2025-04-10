import { useState, useEffect } from "react";
import axios from "axios";
import { toast } from "react-toastify";
import {
  Copy,
  Edit,
  Eye,
  EyeOff,
  Plus,
  Trash2,
  ExternalLink,
} from "lucide-react";
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from "./components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./components/ui/select";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { Label } from "./components/ui/label";
import { Badge } from "./components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./components/ui/card";

// Webhook interface
interface Webhook {
  id: number;
  name: string;
  description: string;
  url: string;
  secret?: string;
  event: string;
  user_id: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

// Initial state for new webhook
const initialWebhookState: Omit<
  Webhook,
  "id" | "user_id" | "created_at" | "updated_at"
> = {
  name: "",
  description: "",
  url: "",
  secret: "",
  event: "deployment",
  active: true,
};

const Webhooks: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook | null>(null);
  const [newWebhook, setNewWebhook] =
    useState<Omit<Webhook, "id" | "user_id" | "created_at" | "updated_at">>(
      initialWebhookState,
    );
  const [showSecret, setShowSecret] = useState(false);

  // Event options
  const eventOptions = [
    { value: "deployment", label: "Deployment" },
    { value: "task", label: "Task Execution" },
  ];

  // Fetch webhooks on component mount
  useEffect(() => {
    fetchWebhooks();
  }, []);

  const fetchWebhooks = async () => {
    setIsLoading(true);
    try {
      const response = await axios.get("/api/webhooks");
      console.log("webhooks", response.data);
      setWebhooks(response.data || []);
    } catch (error) {
      console.error("Failed to fetch webhooks:", error);
      toast.error("Failed to fetch webhooks");
    } finally {
      setIsLoading(false);
    }
  };

  // Handle add webhook
  const handleAddWebhook = async () => {
    try {
      await axios.post("/api/webhooks", newWebhook);
      toast.success("Webhook added successfully");
      setIsAddDialogOpen(false);
      setNewWebhook(initialWebhookState);
      fetchWebhooks();
    } catch (error) {
      console.error("Failed to add webhook:", error);
      toast.error("Failed to add webhook");
    }
  };

  // Handle edit webhook
  const handleEditWebhook = async () => {
    if (!selectedWebhook) return;

    try {
      await axios.put(`/api/webhooks/${selectedWebhook.id}`, selectedWebhook);
      toast.success("Webhook updated successfully");
      setIsEditDialogOpen(false);
      fetchWebhooks();
    } catch (error) {
      console.error("Failed to update webhook:", error);
      toast.error("Failed to update webhook");
    }
  };

  // Handle delete webhook
  const handleDeleteWebhook = async () => {
    if (!selectedWebhook) return;

    try {
      await axios.delete(`/api/webhooks/${selectedWebhook.id}`);
      toast.success("Webhook deleted successfully");
      setIsDeleteDialogOpen(false);
      fetchWebhooks();
    } catch (error) {
      console.error("Failed to delete webhook:", error);
      toast.error("Failed to delete webhook");
    }
  };

  // Copy webhook URL
  const copyWebhookUrl = (webhookId: number) => {
    const url = `${window.location.origin}/webhooks/incoming/${webhookId}`;
    navigator.clipboard.writeText(url);
    toast.success("Webhook URL copied to clipboard");
  };

  // Format webhook event for display
  const formatEvent = (event: string) => {
    return event.charAt(0).toUpperCase() + event.slice(1);
  };

  return (
    <main className="flex-1 p-6 overflow-auto">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-2xl font-inter">Webhooks</h2>
          <p className="text-gray-600">
            Manage external integrations with webhooks
          </p>
        </div>
        <Button onClick={() => setIsAddDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" /> Add Webhook
        </Button>
      </div>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>What are Webhooks?</CardTitle>
          <CardDescription>
            Webhooks allow NyatiCtl to send notifications to external systems
            when events occur, and receive notifications from external systems
            to trigger actions.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid md:grid-cols-2 gap-4">
            <div className="border rounded-md p-4">
              <h3 className="font-medium mb-2">Outgoing Webhooks</h3>
              <p className="text-sm text-gray-600 mb-2">
                NyatiCtl sends HTTP POST requests to your specified URL when
                events like deployments or task executions occur.
              </p>
              <ul className="list-disc list-inside text-sm text-gray-600">
                <li>Deployment notifications</li>
                <li>Task execution results</li>
                <li>HMAC signature for security</li>
              </ul>
            </div>
            <div className="border rounded-md p-4">
              <h3 className="font-medium mb-2">Incoming Webhooks</h3>
              <p className="text-sm text-gray-600 mb-2">
                External services can trigger actions in NyatiCtl by sending
                HTTP POST requests to your webhook URL.
              </p>
              <ul className="list-disc list-inside text-sm text-gray-600">
                <li>Trigger deployments from GitHub/GitLab</li>
                <li>Integrate with CI/CD pipelines</li>
                <li>Connect with third-party services</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {isLoading ? (
        <div className="text-center py-10">
          <p>Loading webhooks...</p>
        </div>
      ) : webhooks.length === 0 ? (
        <div className="text-center py-10 border rounded-md bg-gray-50">
          <h3 className="font-medium text-lg mb-2">No webhooks yet</h3>
          <p className="text-gray-600 mb-4">
            Create your first webhook to start integrating with external
            services
          </p>
          <Button onClick={() => setIsAddDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" /> Add Webhook
          </Button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>Event</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {webhooks &&
                webhooks.map((webhook) => (
                  <TableRow key={webhook.id}>
                    <TableCell className="font-medium">
                      {webhook.name}
                    </TableCell>
                    <TableCell className="max-w-xs truncate">
                      <div className="flex items-center">
                        <span className="truncate">{webhook.url}</span>
                        <ExternalLink className="ml-2 h-4 w-4 text-gray-400" />
                      </div>
                    </TableCell>
                    <TableCell>{formatEvent(webhook.event)}</TableCell>
                    <TableCell>
                      <Badge variant={webhook.active ? "success" : "secondary"}>
                        {webhook.active ? "Active" : "Inactive"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {new Date(webhook.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end space-x-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => copyWebhookUrl(webhook.id)}
                        >
                          <Copy className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedWebhook(webhook);
                            setIsEditDialogOpen(true);
                          }}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedWebhook(webhook);
                            setIsDeleteDialogOpen(true);
                          }}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Add Webhook Dialog */}
      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Webhook</DialogTitle>
            <DialogDescription>
              Create a new webhook to integrate with external services
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="My Webhook"
                value={newWebhook.name}
                onChange={(e) =>
                  setNewWebhook({ ...newWebhook, name: e.target.value })
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="description">Description (Optional)</Label>
              <Input
                id="description"
                placeholder="Notifies my CI/CD system"
                value={newWebhook.description}
                onChange={(e) =>
                  setNewWebhook({ ...newWebhook, description: e.target.value })
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="url">URL</Label>
              <Input
                id="url"
                placeholder="https://example.com/webhook"
                value={newWebhook.url}
                onChange={(e) =>
                  setNewWebhook({ ...newWebhook, url: e.target.value })
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="secret">
                Secret Key (Optional)
                <Button
                  variant="ghost"
                  size="sm"
                  className="ml-2"
                  onClick={() => setShowSecret(!showSecret)}
                >
                  {showSecret ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </Label>
              <Input
                id="secret"
                type={showSecret ? "text" : "password"}
                placeholder="Secret key for HMAC signature verification"
                value={newWebhook.secret}
                onChange={(e) =>
                  setNewWebhook({ ...newWebhook, secret: e.target.value })
                }
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="event">Event</Label>
              <Select
                value={newWebhook.event}
                onValueChange={(value) =>
                  setNewWebhook({ ...newWebhook, event: value })
                }
              >
                <SelectTrigger id="event">
                  <SelectValue placeholder="Select an event" />
                </SelectTrigger>
                <SelectContent>
                  {eventOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="active"
                checked={newWebhook.active}
                onChange={(e) =>
                  setNewWebhook({ ...newWebhook, active: e.target.checked })
                }
                className="rounded border-gray-300"
              />
              <Label htmlFor="active">Active</Label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsAddDialogOpen(false);
                setNewWebhook(initialWebhookState);
              }}
            >
              Cancel
            </Button>
            <Button onClick={handleAddWebhook}>Add Webhook</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Webhook Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Webhook</DialogTitle>
            <DialogDescription>Update webhook details</DialogDescription>
          </DialogHeader>
          {selectedWebhook && (
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-name">Name</Label>
                <Input
                  id="edit-name"
                  placeholder="My Webhook"
                  value={selectedWebhook.name}
                  onChange={(e) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      name: e.target.value,
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-description">Description (Optional)</Label>
                <Input
                  id="edit-description"
                  placeholder="Notifies my CI/CD system"
                  value={selectedWebhook.description}
                  onChange={(e) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      description: e.target.value,
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-url">URL</Label>
                <Input
                  id="edit-url"
                  placeholder="https://example.com/webhook"
                  value={selectedWebhook.url}
                  onChange={(e) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      url: e.target.value,
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-secret">
                  Secret Key (Optional)
                  <Button
                    variant="ghost"
                    size="sm"
                    className="ml-2"
                    onClick={() => setShowSecret(!showSecret)}
                  >
                    {showSecret ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </Button>
                </Label>
                <Input
                  id="edit-secret"
                  type={showSecret ? "text" : "password"}
                  placeholder="Leave empty to keep existing secret"
                  value={selectedWebhook.secret || ""}
                  onChange={(e) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      secret: e.target.value,
                    })
                  }
                />
                <p className="text-xs text-gray-500 mt-1">
                  Leave empty to keep the existing secret
                </p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-event">Event</Label>
                <Select
                  value={selectedWebhook.event}
                  onValueChange={(value) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      event: value,
                    })
                  }
                >
                  <SelectTrigger id="edit-event">
                    <SelectValue placeholder="Select an event" />
                  </SelectTrigger>
                  <SelectContent>
                    {eventOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="edit-active"
                  checked={selectedWebhook.active}
                  onChange={(e) =>
                    setSelectedWebhook({
                      ...selectedWebhook,
                      active: e.target.checked,
                    })
                  }
                  className="rounded border-gray-300"
                />
                <Label htmlFor="edit-active">Active</Label>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsEditDialogOpen(false);
                setSelectedWebhook(null);
              }}
            >
              Cancel
            </Button>
            <Button onClick={handleEditWebhook}>Save Changes</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Webhook Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Webhook</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this webhook? This action cannot
              be undone.
            </DialogDescription>
          </DialogHeader>
          {selectedWebhook && (
            <div className="py-4">
              <p>
                <strong>Name:</strong> {selectedWebhook.name}
              </p>
              <p>
                <strong>URL:</strong> {selectedWebhook.url}
              </p>
              <p>
                <strong>Event:</strong> {formatEvent(selectedWebhook.event)}
              </p>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteDialogOpen(false);
                setSelectedWebhook(null);
              }}
            >
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDeleteWebhook}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
};

export default Webhooks;
