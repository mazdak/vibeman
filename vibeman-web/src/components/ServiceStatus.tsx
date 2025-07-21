import React from "react";
import {
  Activity,
  Server,
  Database,
  Globe,
  AlertCircle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Play,
  Square,
  RotateCw,
  ExternalLink,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
// Service type from generated API
import type { ServerService } from "../generated/api/types.gen";
import { Button } from "./ui/button";

interface ServiceStatusProps {
  services: ServerService[];
  onStartService: (name: string) => void;
  onStopService: (name: string) => void;
  onRestartService: (name: string) => void;
  loading?: boolean;
  error?: string | null;
}

export const ServiceStatus: React.FC<ServiceStatusProps> = ({
  services,
  onStartService,
  onStopService,
  onRestartService,
  loading,
  error,
}) => {
  const getServiceIcon = (name: string) => {
    // Map common service names to icons
    if (name.includes("postgres") || name.includes("mysql") || name.includes("mongo")) {
      return <Database className="w-5 h-5" />;
    }
    if (name.includes("redis") || name.includes("cache")) {
      return <Server className="w-5 h-5" />;
    }
    if (name.includes("nginx") || name.includes("proxy")) {
      return <Globe className="w-5 h-5" />;
    }
    return <Activity className="w-5 h-5" />;
  };

  const getStatusIcon = (status: ServerService["status"]) => {
    switch (status) {
      case "running":
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "starting":
      case "stopping":
        return <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />;
      case "error":
        return <XCircle className="w-4 h-4 text-red-500" />;
      case "stopped":
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  const getStatusColor = (status: ServerService["status"]) => {
    switch (status) {
      case "running":
        return "bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400";
      case "starting":
      case "stopping":
        return "bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400";
      case "error":
        return "bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400";
      case "stopped":
      default:
        return "bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400";
    }
  };


  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="w-6 h-6 text-cyan-500 animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-400">
        <div className="flex items-center gap-2">
          <XCircle className="w-5 h-5" />
          <span>{error}</span>
        </div>
      </div>
    );
  }

  if (services.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500 dark:text-slate-400">
        <Server className="w-8 h-8 mx-auto mb-2 opacity-50" />
        <p>No services configured</p>
        <p className="text-xs mt-2">
          Services can be managed via the CLI with `vibeman service` commands
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <AnimatePresence>
        {services.map((service) => (
          <motion.div
            key={service.name}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="bg-white/50 dark:bg-slate-800/50 backdrop-blur-sm rounded-lg border border-slate-200 dark:border-slate-700 p-4"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="text-slate-600 dark:text-slate-400">
                  {getServiceIcon(service.name)}
                </div>
                <div>
                  <h4 className="font-medium text-slate-900 dark:text-slate-100">
                    {service.name}
                  </h4>
                  <div className="flex items-center gap-4 text-sm text-slate-600 dark:text-slate-400 mt-1">
                    <div className="flex items-center gap-1">
                      {getStatusIcon(service.status)}
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(service.status)}`}>
                        {service.status}
                      </span>
                    </div>
                  </div>
                  {service.port && service.port > 0 && (
                    <div className="flex items-center gap-2 mt-2">
                      <a
                        href={`http://localhost:${service.port}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2 py-1 bg-slate-100 dark:bg-slate-700 rounded text-xs text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
                      >
                        <ExternalLink className="w-3 h-3" />
                        :{service.port}
                      </a>
                    </div>
                  )}
                </div>
              </div>
              
              <div className="flex items-center gap-2">
                {service.status === "stopped" && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onStartService(service.id || service.name || '')}
                    className="text-green-600 hover:text-green-700 hover:bg-green-50 dark:text-green-400 dark:hover:text-green-300 dark:hover:bg-green-900/20"
                  >
                    <Play className="w-4 h-4 mr-1" />
                    Start
                  </Button>
                )}
                {service.status === "running" && (
                  <>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => onRestartService(service.id || service.name || '')}
                      className="text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:text-blue-400 dark:hover:text-blue-300 dark:hover:bg-blue-900/20"
                    >
                      <RotateCw className="w-4 h-4 mr-1" />
                      Restart
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => onStopService(service.id || service.name || '')}
                      className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/20"
                    >
                      <Square className="w-4 h-4 mr-1" />
                      Stop
                    </Button>
                  </>
                )}
                {(service.status === "starting" || service.status === "stopping") && (
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled
                    className="opacity-50"
                  >
                    <RefreshCw className="w-4 h-4 mr-1 animate-spin" />
                    {service.status === "starting" ? "Starting..." : "Stopping..."}
                  </Button>
                )}
              </div>
            </div>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
};