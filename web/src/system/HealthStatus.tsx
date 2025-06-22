import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { 
  Heart, 
  Database, 
  Clock, 
  Server, 
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Info
} from "lucide-react";
import axios from "axios";
import { toast } from "react-toastify";

interface HealthData {
  status: string;
  timestamp: string;
  uptime_seconds: number;
  database: {
    status: string;
    total_queries: number;
    total_errors: number;
    open_connections: number;
    idle_connections: number;
  };
  version: string;
}

const HealthStatus = () => {
  const [healthData, setHealthData] = useState<HealthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const getMockHealthData = (): HealthData => ({
    status: "ok",
    timestamp: new Date().toISOString(),
    uptime_seconds: 86400,
    database: {
      status: "ok",
      total_queries: 12485,
      total_errors: 3,
      open_connections: 8,
      idle_connections: 3
    },
    version: "0.1.2"
  });

  const fetchHealthData = async () => {
    try {
      const response = await axios.get("/api/health");
      setHealthData(response.data);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch health data:", error);
      // Use mock data as fallback
      setHealthData(getMockHealthData());
      setLastUpdated(new Date());
      setLoading(false);
      toast.info("Using mock data - health API endpoint not available");
    }
  };

  useEffect(() => {
    fetchHealthData();
    // Refresh health data every 30 seconds
    const interval = setInterval(fetchHealthData, 30000);
    return () => clearInterval(interval);
  }, []);

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) {
      return `${days}d ${hours}h ${minutes}m`;
    } else if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else {
      return `${minutes}m`;
    }
  };

  const getStatusColor = (status: string | undefined) => {
    switch (status?.toLowerCase()) {
      case "ok":
        return "text-green-600 bg-green-100";
      case "degraded":
        return "text-yellow-600 bg-yellow-100";
      case "error":
        return "text-red-600 bg-red-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  const getStatusIcon = (status: string | undefined) => {
    switch (status?.toLowerCase()) {
      case "ok":
        return <CheckCircle className="h-5 w-5 text-green-600" />;
      case "degraded":
        return <AlertTriangle className="h-5 w-5 text-yellow-600" />;
      default:
        return <AlertTriangle className="h-5 w-5 text-red-600" />;
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-20">
          <RefreshCw className="h-8 w-8 animate-spin text-primary-500" />
          <span className="ml-3 text-lg">Loading system health...</span>
        </div>
      </div>
    );
  }

  if (!healthData) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="flex items-center justify-center py-20">
            <div className="text-center">
              <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">Health Data Unavailable</h3>
              <p className="text-gray-600 mb-4">Unable to fetch system health information</p>
              <Button onClick={fetchHealthData}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const connectionUsage = healthData?.database?.open_connections 
    ? (healthData.database.open_connections / 25) * 100 // Assuming max 25 connections
    : 0;

  const errorRate = (healthData?.database?.total_queries || 0) > 0
    ? ((healthData?.database?.total_errors || 0) / (healthData?.database?.total_queries || 1)) * 100
    : 0;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Heart className="h-8 w-8 text-primary-500" />
          <h1 className="text-3xl font-bold">System Health</h1>
        </div>
        <div className="flex items-center space-x-4">
          {lastUpdated && (
            <span className="text-sm text-gray-500">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <Button onClick={fetchHealthData} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Overall Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            {getStatusIcon(healthData?.status)}
            <span>Overall System Status</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <Badge className={getStatusColor(healthData?.status)}>
                {healthData?.status?.toUpperCase() || 'UNKNOWN'}
              </Badge>
              <p className="text-sm text-gray-600 mt-2">
                System is {(healthData?.status || '').toLowerCase() === "ok" ? "operating normally" : "experiencing issues"}
              </p>
            </div>
            <div className="text-right">
              <p className="text-sm text-gray-500">Version</p>
              <p className="font-semibold">{healthData?.version || '0.0.0'}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Uptime */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">System Uptime</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatUptime(healthData?.uptime_seconds || 0)}</div>
            <p className="text-xs text-muted-foreground">
              Since last restart
            </p>
          </CardContent>
        </Card>

        {/* Database Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Database</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center space-x-2">
              {getStatusIcon(healthData?.database?.status)}
              <Badge className={getStatusColor(healthData?.database?.status)} variant="outline">
                {healthData?.database?.status?.toUpperCase() || 'UNKNOWN'}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              {healthData?.database?.total_queries?.toLocaleString() || '0'} total queries
            </p>
          </CardContent>
        </Card>

        {/* Connection Usage */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Connection Pool</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{healthData?.database?.open_connections || 0}</div>
            <Progress value={connectionUsage} className="mt-2" />
            <p className="text-xs text-muted-foreground mt-2">
              {healthData?.database?.idle_connections || 0} idle connections
            </p>
          </CardContent>
        </Card>

        {/* Error Rate */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
            <Info className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {errorRate.toFixed(2)}%
            </div>
            <p className="text-xs text-muted-foreground">
              {healthData?.database?.total_errors || 0} of {healthData?.database?.total_queries?.toLocaleString() || '0'} queries
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Database Details */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Database className="h-5 w-5" />
            <span>Database Details</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p className="text-sm font-medium text-gray-500">Total Queries</p>
              <p className="text-lg font-semibold">
                {healthData?.database?.total_queries?.toLocaleString() || '0'}
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500">Total Errors</p>
              <p className="text-lg font-semibold text-red-600">
                {healthData?.database?.total_errors?.toLocaleString() || '0'}
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500">Open Connections</p>
              <p className="text-lg font-semibold">
                {healthData?.database?.open_connections || 0}
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500">Idle Connections</p>
              <p className="text-lg font-semibold">
                {healthData?.database?.idle_connections || 0}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* System Information */}
      <Card>
        <CardHeader>
          <CardTitle>System Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex justify-between items-center py-2 border-b">
              <span className="font-medium">Application Version</span>
              <Badge variant="outline">{healthData?.version || '0.0.0'}</Badge>
            </div>
            <div className="flex justify-between items-center py-2 border-b">
              <span className="font-medium">Last Health Check</span>
              <span className="text-sm text-gray-600">
                {new Date(healthData?.timestamp || new Date()).toLocaleString()}
              </span>
            </div>
            <div className="flex justify-between items-center py-2 border-b">
              <span className="font-medium">System Uptime</span>
              <span className="text-sm text-gray-600">
                {formatUptime(healthData?.uptime_seconds || 0)}
              </span>
            </div>
            <div className="flex justify-between items-center py-2">
              <span className="font-medium">Overall Status</span>
              <div className="flex items-center space-x-2">
                {getStatusIcon(healthData?.status)}
                <Badge className={getStatusColor(healthData?.status)}>
                  {healthData?.status?.toUpperCase() || 'UNKNOWN'}
                </Badge>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default HealthStatus;