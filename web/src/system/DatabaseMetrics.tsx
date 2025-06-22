import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { 
  Database, 
  Activity, 
  Clock, 
  Zap, 
  AlertTriangle,
  RefreshCw,
  TrendingUp,
  Server,
  BarChart3
} from "lucide-react";
import axios from "axios";
import { toast } from "react-toastify";

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

const DatabaseMetrics = () => {
  const [metricsData, setMetricsData] = useState<DatabaseMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

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

  const fetchMetricsData = async () => {
    try {
      const response = await axios.get("/api/metrics/database");
      setMetricsData(response.data);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch database metrics:", error);
      // Use mock data as fallback
      setMetricsData(getMockMetricsData());
      setLastUpdated(new Date());
      setLoading(false);
      toast.info("Using mock data - metrics API endpoint not available");
    }
  };

  useEffect(() => {
    fetchMetricsData();
    
    if (autoRefresh) {
      // Refresh metrics every 10 seconds when auto-refresh is enabled
      const interval = setInterval(fetchMetricsData, 10000);
      return () => clearInterval(interval);
    }
  }, [autoRefresh]);

  const getPerformanceStatus = (avgDuration: number) => {
    if (avgDuration < 10) return { status: "excellent", color: "text-green-600 bg-green-100" };
    if (avgDuration < 50) return { status: "good", color: "text-blue-600 bg-blue-100" };
    if (avgDuration < 100) return { status: "moderate", color: "text-yellow-600 bg-yellow-100" };
    return { status: "slow", color: "text-red-600 bg-red-100" };
  };

  const getErrorRateStatus = (errorRate: number) => {
    if (errorRate === 0) return { status: "excellent", color: "text-green-600 bg-green-100" };
    if (errorRate < 1) return { status: "good", color: "text-blue-600 bg-blue-100" };
    if (errorRate < 5) return { status: "moderate", color: "text-yellow-600 bg-yellow-100" };
    return { status: "high", color: "text-red-600 bg-red-100" };
  };

  const getConnectionUsage = (open: number) => {
    const maxConnections = 25; // Based on our configuration
    const usage = (open / maxConnections) * 100;
    return {
      percentage: Math.min(usage, 100),
      status: usage > 80 ? "high" : usage > 60 ? "moderate" : "normal"
    };
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-20">
          <RefreshCw className="h-8 w-8 animate-spin text-primary-500" />
          <span className="ml-3 text-lg">Loading database metrics...</span>
        </div>
      </div>
    );
  }

  if (!metricsData) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="flex items-center justify-center py-20">
            <div className="text-center">
              <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">Metrics Unavailable</h3>
              <p className="text-gray-600 mb-4">Unable to fetch database metrics</p>
              <Button onClick={fetchMetricsData}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const metrics = metricsData.database_metrics;
  const performanceStatus = getPerformanceStatus(metrics.average_duration_ms);
  const errorRateStatus = getErrorRateStatus(metrics.error_rate_percent);
  const connectionUsage = getConnectionUsage(metrics.open_connections);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Database className="h-8 w-8 text-primary-500" />
          <h1 className="text-3xl font-bold">Database Metrics</h1>
        </div>
        <div className="flex items-center space-x-4">
          {lastUpdated && (
            <span className="text-sm text-gray-500">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <Button
            onClick={() => setAutoRefresh(!autoRefresh)}
            variant={autoRefresh ? "default" : "outline"}
            size="sm"
          >
            <Activity className="h-4 w-4 mr-2" />
            Auto Refresh {autoRefresh ? "On" : "Off"}
          </Button>
          <Button onClick={fetchMetricsData} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Key Performance Indicators */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Queries */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Queries</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics.total_queries.toLocaleString()}
            </div>
            <p className="text-xs text-muted-foreground">
              All-time query count
            </p>
          </CardContent>
        </Card>

        {/* Average Response Time */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Response Time</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics.average_duration_ms.toFixed(2)}ms
            </div>
            <Badge className={performanceStatus.color} variant="outline">
              {performanceStatus.status}
            </Badge>
          </CardContent>
        </Card>

        {/* Error Rate */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics.error_rate_percent.toFixed(2)}%
            </div>
            <Badge className={errorRateStatus.color} variant="outline">
              {errorRateStatus.status}
            </Badge>
          </CardContent>
        </Card>

        {/* Active Connections */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Connections</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.open_connections}</div>
            <Progress value={connectionUsage.percentage} className="mt-2" />
            <p className="text-xs text-muted-foreground mt-2">
              {connectionUsage.percentage.toFixed(1)}% of 25 max
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Connection Pool Details */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Server className="h-5 w-5" />
              <span>Connection Pool Status</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Open Connections</span>
              <span className="text-lg font-semibold">{metrics.open_connections}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Idle Connections</span>
              <span className="text-lg font-semibold text-blue-600">{metrics.idle_connections}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Active Connections</span>
              <span className="text-lg font-semibold text-green-600">
                {metrics.open_connections - metrics.idle_connections}
              </span>
            </div>
            <div className="mt-4">
              <div className="flex justify-between text-sm mb-2">
                <span>Pool Usage</span>
                <span>{connectionUsage.percentage.toFixed(1)}%</span>
              </div>
              <Progress value={connectionUsage.percentage} />
              <p className="text-xs text-gray-500 mt-2">
                Maximum pool size: 25 connections
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Performance Metrics */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Zap className="h-5 w-5" />
              <span>Performance Overview</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Total Queries</span>
              <span className="text-lg font-semibold">
                {metrics.total_queries.toLocaleString()}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Total Errors</span>
              <span className="text-lg font-semibold text-red-600">
                {metrics.total_errors.toLocaleString()}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Success Rate</span>
              <span className="text-lg font-semibold text-green-600">
                {(100 - metrics.error_rate_percent).toFixed(2)}%
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Avg Response Time</span>
              <div className="text-right">
                <span className="text-lg font-semibold">
                  {metrics.average_duration_ms.toFixed(2)}ms
                </span>
                <Badge className={`ml-2 ${performanceStatus.color}`} variant="outline">
                  {performanceStatus.status}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Performance Recommendations */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <TrendingUp className="h-5 w-5" />
            <span>Performance Insights</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {metrics.error_rate_percent > 5 && (
              <div className="flex items-start space-x-3 p-3 bg-red-50 rounded-lg">
                <AlertTriangle className="h-5 w-5 text-red-500 mt-0.5" />
                <div>
                  <h4 className="font-medium text-red-800">High Error Rate Detected</h4>
                  <p className="text-sm text-red-600">
                    Current error rate of {metrics.error_rate_percent.toFixed(2)}% is above recommended threshold. 
                    Consider investigating recent queries or database health.
                  </p>
                </div>
              </div>
            )}
            
            {connectionUsage.percentage > 80 && (
              <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
                <AlertTriangle className="h-5 w-5 text-yellow-500 mt-0.5" />
                <div>
                  <h4 className="font-medium text-yellow-800">High Connection Usage</h4>
                  <p className="text-sm text-yellow-600">
                    Connection pool is {connectionUsage.percentage.toFixed(1)}% utilized. 
                    Consider monitoring for potential connection leaks or increasing pool size.
                  </p>
                </div>
              </div>
            )}

            {metrics.average_duration_ms > 100 && (
              <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
                <Clock className="h-5 w-5 text-yellow-500 mt-0.5" />
                <div>
                  <h4 className="font-medium text-yellow-800">Slow Query Performance</h4>
                  <p className="text-sm text-yellow-600">
                    Average response time of {metrics.average_duration_ms.toFixed(2)}ms is higher than optimal. 
                    Consider optimizing database queries or adding indexes.
                  </p>
                </div>
              </div>
            )}

            {metrics.error_rate_percent <= 1 && connectionUsage.percentage < 60 && metrics.average_duration_ms < 50 && (
              <div className="flex items-start space-x-3 p-3 bg-green-50 rounded-lg">
                <Zap className="h-5 w-5 text-green-500 mt-0.5" />
                <div>
                  <h4 className="font-medium text-green-800">Excellent Performance</h4>
                  <p className="text-sm text-green-600">
                    Database is performing optimally with low error rates, good response times, 
                    and healthy connection usage.
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Timestamp Info */}
      <Card>
        <CardContent className="pt-6">
          <div className="text-center text-sm text-gray-500">
            Metrics collected at: {new Date(metricsData.timestamp).toLocaleString()}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default DatabaseMetrics;