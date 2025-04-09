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
import { Button } from "./components/ui/button";
import { Badge } from "./components/ui/badge";
import { ArrowRight, Plus } from "lucide-react";
import { ConfigEntry, User as UserType } from "./App";
import DataTable, { Column, Slot } from "./components/table/DataTable";

// Define dashboard stats interface
interface DashboardStats {
  totalConfigs: number;
  deployed: number;
  draft: number;
  template: number;
}

const CONFIG_COLUMNS: Column[] = [
  {
    key: "name",
    label: "Name",
    render: (_value, row) => (
      <span className="font-medium">{row.name || row.path}</span>
    ),
  },
  {
    key: "status",
    label: "Status",
    render: (_value, row) => (
      <Badge
        variant={
          row.status === "DEPLOYED"
            ? "success"
            : row.status === "DRAFT"
              ? "secondary"
              : "warning"
        }
      >
        {row.status}
      </Badge>
    ),
  },
  {
    key: "path",
    label: "Path",
  },
  {
    key: "actions",
    label: "Action",
    width: "80px",
    align: "center",
  },
];

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

  // Pagination params
  const [params, setParams] = useState({
    page: 1,
    perPage: 5,
    total: 0,
  });

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

  const handleConfigClick = (configPath: string) => {
    navigate(`/configs/${encodeURIComponent(configPath)}`);
  };

  const handleAddConfig = () => {
    navigate("/configs/new");
  };

  // Handle pagination
  const handlePagination = (newParams: typeof params) => {
    setParams(newParams);
    // In a real app, you would fetch the data for the new page
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
              <div className="bg-white rounded-lg shadow">
                <DataTable
                  columns={CONFIG_COLUMNS}
                  items={configs}
                  size="small"
                  showPagination={configs.length > params.perPage}
                  params={params}
                  onPagination={handlePagination}
                >
                  {({ row }) => (
                    <Slot name="actions">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleConfigClick(row.path);
                        }}
                      >
                        <ArrowRight className="h-4 w-4" />
                      </Button>
                    </Slot>
                  )}
                </DataTable>
              </div>
              {configs.length > 5 && (
                <CardFooter className="flex justify-center border-t p-2">
                  <Button variant="link" onClick={() => navigate("/configs")}>
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
                  <Button variant="outline" onClick={() => navigate("/tasks")}>
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
  );
};

export default Dashboard;
