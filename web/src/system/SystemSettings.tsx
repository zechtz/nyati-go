import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { 
  Settings, 
  Database, 
  Zap, 
  Server, 
  Save,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Info
} from "lucide-react";
import { toast } from "react-toastify";

interface SystemConfig {
  database: {
    max_connections: number;
    idle_connections: number;
    connection_lifetime_seconds: number;
    idle_timeout_seconds: number;
  };
  logging: {
    level: string;
    structured_logging: boolean;
  };
  server: {
    port: string;
    request_timeout_seconds: number;
    shutdown_timeout_seconds: number;
  };
}

const SystemSettings = () => {
  const [config, setConfig] = useState<SystemConfig>({
    database: {
      max_connections: 25,
      idle_connections: 5,
      connection_lifetime_seconds: 300,
      idle_timeout_seconds: 60,
    },
    logging: {
      level: "INFO",
      structured_logging: false,
    },
    server: {
      port: "8080",
      request_timeout_seconds: 30,
      shutdown_timeout_seconds: 10,
    },
  });
  
  const [loading, setLoading] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  const logLevels = ["DEBUG", "INFO", "WARN", "ERROR", "FATAL"];

  useEffect(() => {
    // Load current configuration
    loadCurrentConfig();
  }, []);

  const loadCurrentConfig = async () => {
    try {
      // Since we don't have a config endpoint yet, we'll use the current system defaults
      // In a real implementation, this would fetch from /api/config
      setLoading(false);
    } catch (error) {
      console.error("Failed to load configuration:", error);
      toast.error("Failed to load current configuration");
      setLoading(false);
    }
  };

  const handleConfigChange = (section: keyof SystemConfig, field: string, value: any) => {
    setConfig(prev => ({
      ...prev,
      [section]: {
        ...prev[section],
        [field]: value
      }
    }));
    setHasChanges(true);
  };

  const validateConfig = () => {
    const errors: string[] = [];

    // Database validation
    if (config.database.max_connections < 1) {
      errors.push("Max connections must be at least 1");
    }
    if (config.database.idle_connections < 0) {
      errors.push("Idle connections cannot be negative");
    }
    if (config.database.idle_connections > config.database.max_connections) {
      errors.push("Idle connections cannot exceed max connections");
    }
    if (config.database.connection_lifetime_seconds < 1) {
      errors.push("Connection lifetime must be at least 1 second");
    }
    if (config.database.idle_timeout_seconds < 0) {
      errors.push("Idle timeout cannot be negative");
    }

    // Server validation
    const port = parseInt(config.server.port);
    if (isNaN(port) || port < 1 || port > 65535) {
      errors.push("Port must be between 1 and 65535");
    }
    if (config.server.request_timeout_seconds < 1) {
      errors.push("Request timeout must be at least 1 second");
    }
    if (config.server.shutdown_timeout_seconds < 1) {
      errors.push("Shutdown timeout must be at least 1 second");
    }

    return errors;
  };

  const saveConfiguration = async () => {
    const errors = validateConfig();
    if (errors.length > 0) {
      toast.error(`Configuration errors: ${errors.join(", ")}`);
      return;
    }

    setLoading(true);
    try {
      // In a real implementation, this would POST to /api/config
      await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate API call
      
      setHasChanges(false);
      toast.success("Configuration saved successfully. Restart required for some changes to take effect.");
    } catch (error) {
      console.error("Failed to save configuration:", error);
      toast.error("Failed to save configuration");
    }
    setLoading(false);
  };

  const resetToDefaults = () => {
    setConfig({
      database: {
        max_connections: 25,
        idle_connections: 5,
        connection_lifetime_seconds: 300,
        idle_timeout_seconds: 60,
      },
      logging: {
        level: "INFO",
        structured_logging: false,
      },
      server: {
        port: "8080",
        request_timeout_seconds: 30,
        shutdown_timeout_seconds: 10,
      },
    });
    setHasChanges(true);
    toast.info("Configuration reset to defaults");
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Settings className="h-8 w-8 text-primary-500" />
          <h1 className="text-3xl font-bold">System Configuration</h1>
        </div>
        <div className="flex items-center space-x-4">
          {hasChanges && (
            <Badge variant="outline" className="text-yellow-600 border-yellow-600">
              Unsaved Changes
            </Badge>
          )}
          <Button onClick={resetToDefaults} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Reset to Defaults
          </Button>
          <Button 
            onClick={saveConfiguration} 
            disabled={!hasChanges || loading}
            size="sm"
          >
            <Save className="h-4 w-4 mr-2" />
            {loading ? "Saving..." : "Save Configuration"}
          </Button>
        </div>
      </div>

      {/* Configuration Notice */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-start space-x-3 p-3 bg-blue-50 rounded-lg">
            <Info className="h-5 w-5 text-blue-500 mt-0.5" />
            <div>
              <h4 className="font-medium text-blue-800">Configuration Management</h4>
              <p className="text-sm text-blue-600">
                This interface allows you to view and modify system configuration. In production, 
                these settings would be managed through environment variables. Some changes require 
                a system restart to take effect.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Database Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Database className="h-5 w-5" />
            <span>Database Configuration</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label htmlFor="max-connections">Max Connections</Label>
              <Input
                id="max-connections"
                type="number"
                min="1"
                max="100"
                value={config.database.max_connections}
                onChange={(e) => handleConfigChange("database", "max_connections", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                Maximum concurrent database connections (Current: 25)
              </p>
            </div>
            <div>
              <Label htmlFor="idle-connections">Idle Connections</Label>
              <Input
                id="idle-connections"
                type="number"
                min="0"
                max={config.database.max_connections}
                value={config.database.idle_connections}
                onChange={(e) => handleConfigChange("database", "idle_connections", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                Number of idle connections to maintain
              </p>
            </div>
            <div>
              <Label htmlFor="connection-lifetime">Connection Lifetime (seconds)</Label>
              <Input
                id="connection-lifetime"
                type="number"
                min="1"
                value={config.database.connection_lifetime_seconds}
                onChange={(e) => handleConfigChange("database", "connection_lifetime_seconds", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                Maximum lifetime of a database connection
              </p>
            </div>
            <div>
              <Label htmlFor="idle-timeout">Idle Timeout (seconds)</Label>
              <Input
                id="idle-timeout"
                type="number"
                min="0"
                value={config.database.idle_timeout_seconds}
                onChange={(e) => handleConfigChange("database", "idle_timeout_seconds", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                Timeout for idle connections
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Logging Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="h-5 w-5" />
            <span>Logging Configuration</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label htmlFor="log-level">Log Level</Label>
              <select
                id="log-level"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                value={config.logging.level}
                onChange={(e) => handleConfigChange("logging", "level", e.target.value)}
              >
                {logLevels.map(level => (
                  <option key={level} value={level}>{level}</option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">
                Minimum log level to output
              </p>
            </div>
            <div>
              <Label htmlFor="structured-logging">Structured Logging</Label>
              <div className="flex items-center space-x-2 mt-2">
                <input
                  id="structured-logging"
                  type="checkbox"
                  checked={config.logging.structured_logging}
                  onChange={(e) => handleConfigChange("logging", "structured_logging", e.target.checked)}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                />
                <span className="text-sm">Enable JSON structured logging</span>
              </div>
              <p className="text-xs text-gray-500 mt-1">
                Output logs in JSON format for better parsing
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Server Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Server className="h-5 w-5" />
            <span>Server Configuration</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <Label htmlFor="server-port">Server Port</Label>
              <Input
                id="server-port"
                type="number"
                min="1"
                max="65535"
                value={config.server.port}
                onChange={(e) => handleConfigChange("server", "port", e.target.value)}
              />
              <p className="text-xs text-gray-500 mt-1">
                HTTP server port (1-65535)
              </p>
            </div>
            <div>
              <Label htmlFor="request-timeout">Request Timeout (seconds)</Label>
              <Input
                id="request-timeout"
                type="number"
                min="1"
                value={config.server.request_timeout_seconds}
                onChange={(e) => handleConfigChange("server", "request_timeout_seconds", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                HTTP request timeout
              </p>
            </div>
            <div>
              <Label htmlFor="shutdown-timeout">Shutdown Timeout (seconds)</Label>
              <Input
                id="shutdown-timeout"
                type="number"
                min="1"
                value={config.server.shutdown_timeout_seconds}
                onChange={(e) => handleConfigChange("server", "shutdown_timeout_seconds", parseInt(e.target.value))}
              />
              <p className="text-xs text-gray-500 mt-1">
                Graceful shutdown timeout
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Environment Variables Information */}
      <Card>
        <CardHeader>
          <CardTitle>Environment Variables</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <p className="text-sm text-gray-600">
              These settings correspond to the following environment variables:
            </p>
            <div className="bg-gray-50 p-4 rounded-lg space-y-2 text-sm font-mono">
              <div>NYATI_DB_MAX_CONNS={config.database.max_connections}</div>
              <div>NYATI_DB_IDLE_CONNS={config.database.idle_connections}</div>
              <div>NYATI_DB_CONN_LIFETIME={config.database.connection_lifetime_seconds}s</div>
              <div>NYATI_DB_IDLE_TIME={config.database.idle_timeout_seconds}s</div>
              <div>NYATI_LOG_LEVEL={config.logging.level}</div>
              <div>NYATI_STRUCTURED_LOGGING={config.logging.structured_logging.toString()}</div>
              <div>NYATI_PORT={config.server.port}</div>
              <div>NYATI_REQUEST_TIMEOUT={config.server.request_timeout_seconds}s</div>
              <div>NYATI_SHUTDOWN_TIMEOUT={config.server.shutdown_timeout_seconds}s</div>
            </div>
            <div className="flex items-start space-x-3 p-3 bg-yellow-50 rounded-lg">
              <AlertTriangle className="h-5 w-5 text-yellow-500 mt-0.5" />
              <div>
                <h4 className="font-medium text-yellow-800">Production Note</h4>
                <p className="text-sm text-yellow-600">
                  In production environments, set these values as environment variables 
                  and restart the application to apply changes.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Save Actions */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              {hasChanges ? (
                <AlertTriangle className="h-5 w-5 text-yellow-500" />
              ) : (
                <CheckCircle className="h-5 w-5 text-green-500" />
              )}
              <span className="text-sm">
                {hasChanges ? "You have unsaved changes" : "Configuration is up to date"}
              </span>
            </div>
            <div className="flex space-x-2">
              <Button 
                onClick={resetToDefaults} 
                variant="outline"
                disabled={loading}
              >
                Reset to Defaults
              </Button>
              <Button 
                onClick={saveConfiguration} 
                disabled={!hasChanges || loading}
              >
                <Save className="h-4 w-4 mr-2" />
                {loading ? "Saving..." : "Save Configuration"}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default SystemSettings;