import { useState, useEffect, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { 
  FileText, 
  Play, 
  Pause, 
  Trash2, 
  Download, 
  Search,
  Filter,
  Eye,
  EyeOff,
  Terminal,
  RefreshCw
} from "lucide-react";
import { toast } from "react-toastify";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  fields?: Record<string, any>;
  source?: string;
}

const SystemLogs = () => {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [filteredLogs, setFilteredLogs] = useState<LogEntry[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [selectedLevels, setSelectedLevels] = useState<Set<string>>(new Set(["INFO", "WARN", "ERROR", "FATAL"]));
  const [searchTerm, setSearchTerm] = useState("");
  const [showStructuredView, setShowStructuredView] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  
  const logsEndRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const logLevels = ["DEBUG", "INFO", "WARN", "ERROR", "FATAL"];

  const logLevelColors = {
    DEBUG: "text-gray-600 bg-gray-100",
    INFO: "text-blue-600 bg-blue-100",
    WARN: "text-yellow-600 bg-yellow-100",
    ERROR: "text-red-600 bg-red-100",
    FATAL: "text-red-800 bg-red-200"
  };

  useEffect(() => {
    // Simulate initial log data
    const initialLogs: LogEntry[] = [
      {
        timestamp: new Date().toISOString(),
        level: "INFO",
        message: "System started successfully",
        fields: { component: "main", version: "0.1.2" }
      },
      {
        timestamp: new Date(Date.now() - 30000).toISOString(),
        level: "INFO",
        message: "Database connection established",
        fields: { component: "database", connections: 5 }
      },
      {
        timestamp: new Date(Date.now() - 60000).toISOString(),
        level: "WARN",
        message: "High memory usage detected",
        fields: { component: "monitor", memory_usage: "85%" }
      }
    ];
    setLogs(initialLogs);
  }, []);

  useEffect(() => {
    // Filter logs based on selected levels and search term
    const filtered = logs.filter(log => {
      const levelMatch = selectedLevels.has(log.level);
      const searchMatch = searchTerm === "" || 
        (log.message?.toLowerCase().includes(searchTerm.toLowerCase()) || false) ||
        (log.fields && JSON.stringify(log.fields).toLowerCase().includes(searchTerm.toLowerCase()));
      
      return levelMatch && searchMatch;
    });
    setFilteredLogs(filtered);
  }, [logs, selectedLevels, searchTerm]);

  useEffect(() => {
    // Auto-scroll to bottom when new logs arrive
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [filteredLogs, autoScroll]);

  const toggleStreaming = () => {
    if (isStreaming) {
      stopStreaming();
    } else {
      startStreaming();
    }
  };

  const startStreaming = () => {
    try {
      // In a real implementation, this would connect to a WebSocket endpoint for live logs
      // For demo purposes, we'll simulate periodic log updates
      const sessionId = Math.random().toString(36).substring(7);
      
      // Simulate WebSocket connection
      const interval = setInterval(() => {
        const newLog: LogEntry = {
          timestamp: new Date().toISOString(),
          level: logLevels[Math.floor(Math.random() * logLevels.length)],
          message: generateRandomLogMessage(),
          fields: {
            component: ["database", "auth", "api", "ssh"][Math.floor(Math.random() * 4)],
            session_id: sessionId
          }
        };
        
        setLogs(prev => [...prev.slice(-99), newLog]); // Keep last 100 logs
      }, 2000);

      wsRef.current = { close: () => clearInterval(interval) } as any;
      setIsStreaming(true);
      toast.success("Started log streaming");
    } catch (error) {
      toast.error("Failed to start log streaming");
    }
  };

  const stopStreaming = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsStreaming(false);
    toast.info("Stopped log streaming");
  };

  const generateRandomLogMessage = () => {
    const messages = [
      "Processing API request",
      "Database query executed successfully",
      "User authentication completed",
      "SSH connection established",
      "Configuration updated",
      "Health check passed",
      "Cache miss for key",
      "Webhook triggered",
      "File uploaded successfully",
      "Background task completed"
    ];
    return messages[Math.floor(Math.random() * messages.length)];
  };

  const toggleLogLevel = (level: string) => {
    const newLevels = new Set(selectedLevels);
    if (newLevels.has(level)) {
      newLevels.delete(level);
    } else {
      newLevels.add(level);
    }
    setSelectedLevels(newLevels);
  };

  const clearLogs = () => {
    setLogs([]);
    toast.success("Logs cleared");
  };

  const exportLogs = () => {
    const logsText = filteredLogs.map(log => {
      if (showStructuredView && log.fields) {
        return JSON.stringify({ ...log, timestamp: log.timestamp });
      } else {
        return `[${log.timestamp}] ${log.level} ${log.message}`;
      }
    }).join('\n');

    const blob = new Blob([logsText], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `system-logs-${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    toast.success("Logs exported successfully");
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <FileText className="h-8 w-8 text-primary-500" />
          <h1 className="text-3xl font-bold">System Logs</h1>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            onClick={toggleStreaming}
            variant={isStreaming ? "destructive" : "default"}
            size="sm"
          >
            {isStreaming ? (
              <>
                <Pause className="h-4 w-4 mr-2" />
                Stop Stream
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-2" />
                Start Stream
              </>
            )}
          </Button>
          <Button onClick={exportLogs} variant="outline" size="sm">
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
          <Button onClick={clearLogs} variant="outline" size="sm">
            <Trash2 className="h-4 w-4 mr-2" />
            Clear
          </Button>
        </div>
      </div>

      {/* Controls */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Filter className="h-5 w-5" />
            <span>Log Filters</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Search */}
          <div className="flex items-center space-x-2">
            <Search className="h-4 w-4 text-gray-500" />
            <Input
              placeholder="Search logs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="max-w-md"
            />
          </div>

          {/* Log Levels */}
          <div className="flex flex-wrap gap-2">
            <span className="text-sm font-medium">Log Levels:</span>
            {logLevels.map(level => (
              <Badge
                key={level}
                className={`cursor-pointer ${
                  selectedLevels.has(level) 
                    ? logLevelColors[level as keyof typeof logLevelColors]
                    : "text-gray-400 bg-gray-100"
                }`}
                onClick={() => toggleLogLevel(level)}
              >
                {level}
              </Badge>
            ))}
          </div>

          {/* View Options */}
          <div className="flex items-center space-x-4">
            <Button
              onClick={() => setShowStructuredView(!showStructuredView)}
              variant="outline"
              size="sm"
            >
              {showStructuredView ? (
                <>
                  <EyeOff className="h-4 w-4 mr-2" />
                  Simple View
                </>
              ) : (
                <>
                  <Eye className="h-4 w-4 mr-2" />
                  Structured View
                </>
              )}
            </Button>
            <Button
              onClick={() => setAutoScroll(!autoScroll)}
              variant="outline"
              size="sm"
            >
              <Terminal className="h-4 w-4 mr-2" />
              Auto Scroll: {autoScroll ? "On" : "Off"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Log Display */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Terminal className="h-5 w-5" />
              <span>Live Logs</span>
              {isStreaming && (
                <Badge className="text-green-600 bg-green-100">
                  <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                  Streaming
                </Badge>
              )}
            </div>
            <span className="text-sm text-gray-500">
              {filteredLogs.length} entries
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="bg-gray-900 text-gray-100 p-4 rounded-lg h-96 overflow-auto font-mono text-sm">
            {filteredLogs.length === 0 ? (
              <div className="text-center text-gray-500 py-8">
                No logs match the current filters
              </div>
            ) : (
              filteredLogs.map((log, index) => (
                <div key={index} className="mb-2">
                  {showStructuredView ? (
                    <div className="space-y-1">
                      <div className="flex items-center space-x-2">
                        <span className="text-gray-400">{formatTimestamp(log.timestamp)}</span>
                        <Badge className={`text-xs ${logLevelColors[log.level as keyof typeof logLevelColors]}`}>
                          {log.level}
                        </Badge>
                        <span>{log.message}</span>
                      </div>
                      {log.fields && Object.keys(log.fields).length > 0 && (
                        <div className="ml-4 text-gray-300">
                          {Object.entries(log.fields).map(([key, value]) => (
                            <div key={key} className="text-xs">
                              <span className="text-blue-400">{key}:</span> {JSON.stringify(value)}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="flex items-start space-x-2">
                      <span className="text-gray-400 w-20">{formatTimestamp(log.timestamp)}</span>
                      <span className={`w-12 text-xs ${
                        log.level === "ERROR" || log.level === "FATAL" ? "text-red-400" :
                        log.level === "WARN" ? "text-yellow-400" :
                        log.level === "INFO" ? "text-blue-400" :
                        "text-gray-400"
                      }`}>
                        {log.level}
                      </span>
                      <span className="flex-1">{log.message}</span>
                    </div>
                  )}
                </div>
              ))
            )}
            <div ref={logsEndRef} />
          </div>
        </CardContent>
      </Card>

      {/* Log Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {logLevels.map(level => {
          const count = logs.filter(log => log.level === level).length;
          return (
            <Card key={level}>
              <CardContent className="pt-6">
                <div className="text-center">
                  <div className="text-2xl font-bold">{count}</div>
                  <Badge className={logLevelColors[level as keyof typeof logLevelColors]} variant="outline">
                    {level}
                  </Badge>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
};

export default SystemLogs;