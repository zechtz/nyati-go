import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  Monitor, 
  Database, 
  Clock, 
  Server, 
  RefreshCw,
  CheckCircle,
  AlertTriangle,
  TrendingUp,
  BarChart3,
  Zap,
  Heart
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

interface DatabaseMetrics {
  database_metrics: {
    total_queries: number;
    total_errors: number;
    average_duration_ms: number;
    open_connections: number;
    idle_connections: number;
    error_rate_percent: number;
  };
  timestamp: string;
}

const SystemOverview = () => {
  const [healthData, setHealthData] = useState<HealthData | null>(null);
  const [metricsData, setMetricsData] = useState<DatabaseMetrics | null>(null);
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

  const getMockMetricsData = (): DatabaseMetrics => ({
    database_metrics: {
      total_queries: 12485,
      total_errors: 3,
      average_duration_ms: 24.5,
      open_connections: 8,
      idle_connections: 3,
      error_rate_percent: 0.02
    },
    timestamp: new Date().toISOString()
  });

  const fetchSystemData = async () => {
    try {
      const [healthResponse, metricsResponse] = await Promise.all([
        axios.get("/api/health").catch(() => ({ data: getMockHealthData() })),
        axios.get("/api/metrics/database").catch(() => ({ data: getMockMetricsData() }))
      ]);
      
      setHealthData(healthResponse.data);
      setMetricsData(metricsResponse.data);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch system data:", error);
      // Use mock data as fallback
      setHealthData(getMockHealthData());
      setMetricsData(getMockMetricsData());
      setLastUpdated(new Date());
      setLoading(false);
      toast.info("Using mock data - API endpoints not available");
    }
  };

  useEffect(() => {
    fetchSystemData();
    // Refresh data every 30 seconds
    const interval = setInterval(fetchSystemData, 30000);
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
      default:
        return "text-red-600 bg-red-100";
    }
  };

  const getStatusIcon = (status: string | undefined) => {
    switch (status?.toLowerCase()) {
      case "ok":
        return <CheckCircle className="h-5 w-5 text-green-600" />;
      default:
        return <AlertTriangle className="h-5 w-5 text-red-600" />;
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-20">
          <RefreshCw className="h-8 w-8 animate-spin text-primary-500" />
          <span className="ml-3 text-lg">Loading system overview...</span>
        </div>
      </div>
    );
  }

  if (!healthData || !metricsData) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="flex items-center justify-center py-20">
            <div className="text-center">
              <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">System Data Unavailable</h3>
              <p className="text-gray-600 mb-4">Unable to fetch system information</p>
              <Button onClick={fetchSystemData}>
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
    ? (healthData.database.open_connections / 25) * 100
    : 0;

  const metrics = metricsData?.database_metrics;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Monitor className="h-8 w-8 text-primary-500" />
          <h1 className="text-3xl font-bold">System Overview</h1>
        </div>
        <div className="flex items-center space-x-4">
          {lastUpdated && (
            <span className="text-sm text-gray-500">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <Button onClick={fetchSystemData} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* System Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Overall Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">System Status</CardTitle>
            <Heart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center space-x-2">
              {getStatusIcon(healthData?.status)}
              <Badge className={getStatusColor(healthData?.status)}>
                {healthData?.status?.toUpperCase() || 'UNKNOWN'}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              v{healthData?.version || '0.0.0'}
            </p>
          </CardContent>
        </Card>

        {/* Uptime */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Uptime</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatUptime(healthData?.uptime_seconds || 0)}</div>
            <p className="text-xs text-muted-foreground">
              Since last restart
            </p>
          </CardContent>
        </Card>

        {/* Database Performance */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Response</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics?.average_duration_ms?.toFixed(1) || '0.0'}ms</div>
            <p className="text-xs text-muted-foreground">
              Query response time
            </p>
          </CardContent>
        </Card>

        {/* Error Rate */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics?.error_rate_percent?.toFixed(2) || '0.00'}%</div>
            <p className="text-xs text-muted-foreground">
              {metrics?.total_errors || 0} of {metrics?.total_queries?.toLocaleString() || '0'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Sections */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="database">Database</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* System Health */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Heart className="h-5 w-5" />
                  <span>System Health</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="font-medium">Overall Status</span>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(healthData?.status)}
                    <Badge className={getStatusColor(healthData?.status)}>
                      {healthData?.status?.toUpperCase() || 'UNKNOWN'}
                    </Badge>
                  </div>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Database Status</span>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(healthData?.database?.status)}
                    <Badge className={getStatusColor(healthData?.database?.status)}>
                      {healthData?.database?.status?.toUpperCase() || 'UNKNOWN'}
                    </Badge>
                  </div>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Version</span>
                  <Badge variant="outline">{healthData?.version || '0.0.0'}</Badge>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Uptime</span>
                  <span className="font-mono">{formatUptime(healthData?.uptime_seconds || 0)}</span>
                </div>
              </CardContent>
            </Card>

            {/* Quick Stats */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <BarChart3 className="h-5 w-5" />
                  <span>Quick Stats</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="font-medium">Total Queries</span>
                  <span className="font-mono">{metrics?.total_queries?.toLocaleString() || '0'}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Success Rate</span>
                  <span className="font-mono text-green-600">
                    {(100 - (metrics?.error_rate_percent || 0)).toFixed(2)}%
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Avg Response</span>
                  <span className="font-mono">{metrics?.average_duration_ms?.toFixed(2) || '0.00'}ms</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Active Connections</span>
                  <span className="font-mono">{metrics?.open_connections || 0}</span>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="database" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Connection Pool */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Server className="h-5 w-5" />
                  <span>Connection Pool</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="font-medium">Open Connections</span>
                  <span className="text-lg font-semibold">{metrics?.open_connections || 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Idle Connections</span>
                  <span className="text-lg font-semibold text-blue-600">{metrics?.idle_connections || 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Active Connections</span>
                  <span className="text-lg font-semibold text-green-600">
                    {(metrics?.open_connections || 0) - (metrics?.idle_connections || 0)}
                  </span>
                </div>
                <div className="mt-4">
                  <div className="flex justify-between text-sm mb-2">
                    <span>Pool Usage</span>
                    <span>{connectionUsage.toFixed(1)}%</span>
                  </div>
                  <Progress value={connectionUsage} />
                </div>
              </CardContent>
            </Card>

            {/* Database Performance */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <Database className="h-5 w-5" />
                  <span>Database Performance</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="font-medium">Total Queries</span>
                  <span className="text-lg font-semibold">{metrics?.total_queries?.toLocaleString() || '0'}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Total Errors</span>
                  <span className="text-lg font-semibold text-red-600">{metrics?.total_errors || 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Error Rate</span>
                  <span className="text-lg font-semibold text-red-600">{metrics?.error_rate_percent?.toFixed(2) || '0.00'}%</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="font-medium">Avg Response Time</span>
                  <span className="text-lg font-semibold">{metrics?.average_duration_ms?.toFixed(2) || '0.00'}ms</span>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <TrendingUp className="h-5 w-5" />
                <span>Performance Analysis</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Performance Status */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="text-center p-4 border rounded-lg">
                    <div className="text-2xl font-bold text-green-600">{(100 - (metrics?.error_rate_percent || 0)).toFixed(1)}%</div>
                    <div className="text-sm text-gray-600">Success Rate</div>
                  </div>
                  <div className="text-center p-4 border rounded-lg">
                    <div className="text-2xl font-bold text-blue-600">{metrics?.average_duration_ms?.toFixed(1) || '0.0'}ms</div>
                    <div className="text-sm text-gray-600">Avg Response Time</div>
                  </div>
                  <div className="text-center p-4 border rounded-lg">
                    <div className="text-2xl font-bold text-purple-600">{connectionUsage.toFixed(1)}%</div>
                    <div className="text-sm text-gray-600">Connection Usage</div>
                  </div>
                </div>

                {/* Performance Insights */}
                <div className="mt-6">
                  <h4 className="font-semibold mb-3">Performance Insights</h4>
                  <div className="space-y-3">
                    {(metrics?.error_rate_percent || 0) === 0 ? (
                      <div className="flex items-start space-x-3 p-3 bg-green-50 rounded-lg">
                        <CheckCircle className="h-5 w-5 text-green-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-green-800">Zero Error Rate</h5>
                          <p className="text-sm text-green-600">All database queries are executing successfully.</p>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
                        <AlertTriangle className="h-5 w-5 text-yellow-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-yellow-800">Error Rate: {metrics?.error_rate_percent?.toFixed(2) || '0.00'}%</h5>
                          <p className="text-sm text-yellow-600">
                            {metrics.total_errors} queries failed out of {metrics.total_queries.toLocaleString()} total.
                          </p>
                        </div>
                      </div>
                    )}

                    {metrics.average_duration_ms < 50 ? (
                      <div className="flex items-start space-x-3 p-3 bg-green-50 rounded-lg">
                        <Zap className="h-5 w-5 text-green-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-green-800">Excellent Response Times</h5>
                          <p className="text-sm text-green-600">
                            Average query response time of {metrics.average_duration_ms.toFixed(2)}ms is excellent.
                          </p>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
                        <Clock className="h-5 w-5 text-yellow-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-yellow-800">Response Time Optimization</h5>
                          <p className="text-sm text-yellow-600">
                            Consider optimizing queries with {metrics.average_duration_ms.toFixed(2)}ms average response time.
                          </p>
                        </div>
                      </div>
                    )}

                    {connectionUsage < 70 ? (
                      <div className="flex items-start space-x-3 p-3 bg-green-50 rounded-lg">
                        <Server className="h-5 w-5 text-green-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-green-800">Healthy Connection Usage</h5>
                          <p className="text-sm text-green-600">
                            Connection pool usage at {connectionUsage.toFixed(1)}% is within normal range.
                          </p>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
                        <AlertTriangle className="h-5 w-5 text-yellow-500 mt-0.5" />
                        <div>
                          <h5 className="font-medium text-yellow-800">High Connection Usage</h5>
                          <p className="text-sm text-yellow-600">
                            Connection pool at {connectionUsage.toFixed(1)}% capacity. Monitor for connection leaks.
                          </p>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default SystemOverview;