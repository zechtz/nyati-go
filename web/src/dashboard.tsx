import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import { toast } from "react-toastify";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./components/ui/table";
import { Button } from "./components/ui/button";
import { Badge } from "./components/ui/badge";
import { Search, User, LogOut, ArrowRight, Plus } from "lucide-react";
import { Input } from "./components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "./components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./components/ui/dropdown-menu";
import { useAuth } from "./contexts/AuthContext";
import Sidebar from "./components/sidebar/Sidebar";
import { ConfigEntry, User as UserType } from "./App";

// Define dashboard stats interface
interface DashboardStats {
  totalConfigs: number;
  deployed: number;
  draft: number;
  template: number;
}

const Dashboard: React.FC = () => {
  const [configs, setConfigs] = useState<ConfigEntry[]>([]);
  const [stats, setStats] = useState<DashboardStats>({
    totalConfigs: 0,
    deployed: 0,
    draft: 0,
    template: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState<UserType>({} as UserType);

  const { logout } = useAuth();
  const navigate = useNavigate();

  const fetchUser = () => {
    const userStore = localStorage.getItem("NYATI_USER");
    if (userStore) {
      const user: UserType = JSON.parse(userStore);
      setUser(user);
    }
  };

  const fetchConfigs = async () => {
    try {
      const response = await axios.get("/api/configs");
      setConfigs(Array.isArray(response.data) ? response.data : []);
    } catch (error) {
      console.error("Failed to fetch configs:", error);
      toast.error("Failed to fetch configurations");
    }
  };

  const fetchDashboardStats = async () => {
    try {
      const response = await axios.get("/api/dashboard/stats");
      setStats(response.data);
    } catch (error) {
      console.error("Failed to fetch dashboard stats:", error);
      toast.error("Failed to fetch dashboard statistics");
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = () => {
    logout();
    toast.success("Logged out successfully!");
    navigate("/login");
  };

  const handleConfigClick = (configPath: string) => {
    navigate(`/configs/${encodeURIComponent(configPath)}`);
  };

  const handleAddConfig = () => {
    navigate("/configs/new");
  };

  useEffect(() => {
    fetchUser();
    fetchConfigs();
    fetchDashboardStats();
  }, []);

  // Status card background colors
  const statusColors = {
    totalConfigs: "bg-blue-50 text-blue-700",
    deployed: "bg-green-50 text-green-700",
    draft: "bg-amber-50 text-amber-700",
    template: "bg-purple-50 text-purple-700",
  };

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Header */}
      <header className="absolute top-0 left-0 right-0 bg-primary-500 text-white p-4 flex justify-between items-center z-10">
        <h1 className="text-2xl font-inter">NyatiCtl</h1>
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
                <span>{user.email}</span>
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
          <div className="mb-6">
            <h2 className="text-2xl font-inter">Dashboard</h2>
            <p className="text-gray-600">Welcome back, {user.email}</p>
          </div>

          {isLoading ? (
            <div className="flex justify-center items-center h-40">
              <p>Loading dashboard data...</p>
            </div>
          ) : (
            <>
              {/* Stats Cards */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                <Card className={statusColors.totalConfigs}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Total Configs</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold">{stats.totalConfigs}</p>
                  </CardContent>
                </Card>
                <Card className={statusColors.deployed}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Deployed</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold">{stats.deployed}</p>
                  </CardContent>
                </Card>
                <Card className={statusColors.draft}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Draft</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold">{stats.draft}</p>
                  </CardContent>
                </Card>
                <Card className={statusColors.template}>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Templates</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-3xl font-bold">{stats.template}</p>
                  </CardContent>
                </Card>
              </div>

              {/* Recent Configs */}
              <div className="mb-6">
                <div className="flex justify-between items-center mb-4">
                  <h3 className="text-xl font-inter">Recent Configurations</h3>
                  <Button size="sm" onClick={handleAddConfig}>
                    <Plus className="h-4 w-4 mr-2" /> Add Config
                  </Button>
                </div>

                <Card>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Path</TableHead>
                        <TableHead>Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {configs.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} className="text-center py-6">
                            <p>No configurations found</p>
                            <Button
                              variant="link"
                              onClick={handleAddConfig}
                              className="mt-2"
                            >
                              Create your first configuration
                            </Button>
                          </TableCell>
                        </TableRow>
                      ) : (
                        configs.slice(0, 5).map((config) => (
                          <TableRow
                            key={config.path}
                            className="cursor-pointer hover:bg-gray-50"
                            onClick={() => handleConfigClick(config.path)}
                          >
                            <TableCell className="font-medium">
                              {config.name || config.path}
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
                            <TableCell className="text-gray-500">
                              {config.path}
                            </TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleConfigClick(config.path);
                                }}
                              >
                                <ArrowRight className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                  {configs.length > 5 && (
                    <CardFooter className="flex justify-center border-t p-2">
                      <Button
                        variant="link"
                        onClick={() => navigate("/configs")}
                      >
                        View all configurations
                      </Button>
                    </CardFooter>
                  )}
                </Card>
              </div>

              {/* Quick Actions */}
              <div className="mb-6">
                <h3 className="text-xl font-inter mb-4">Quick Actions</h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Deploy Config</CardTitle>
                      <CardDescription>
                        Deploy your configurations to your servers
                      </CardDescription>
                    </CardHeader>
                    <CardFooter>
                      <Button
                        variant="outline"
                        onClick={() => navigate("/configs")}
                      >
                        Go to Configs
                      </Button>
                    </CardFooter>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>Run Tasks</CardTitle>
                      <CardDescription>
                        Execute tasks on your remote servers
                      </CardDescription>
                    </CardHeader>
                    <CardFooter>
                      <Button
                        variant="outline"
                        onClick={() => navigate("/tasks")}
                      >
                        Go to Tasks
                      </Button>
                    </CardFooter>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>View Documentation</CardTitle>
                      <CardDescription>
                        Read the documentation for NyatiCtl
                      </CardDescription>
                    </CardHeader>
                    <CardFooter>
                      <Button
                        variant="outline"
                        onClick={() =>
                          window.open("https://docs.example.com", "_blank")
                        }
                      >
                        Open Docs
                      </Button>
                    </CardFooter>
                  </Card>
                </div>
              </div>
            </>
          )}
        </main>
      </div>
    </div>
  );
};

export default Dashboard;
