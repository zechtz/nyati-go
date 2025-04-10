import React from "react";
import EnvironmentManager from "./EnvironmentManager";

const EnvironmentsPage: React.FC = () => {
  return (
    <main className="flex-1 p-6 overflow-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-inter">Environment Variables</h2>
        <p className="text-gray-600">
          Manage environment variables for different deployment targets.
        </p>
      </div>

      <EnvironmentManager />
    </main>
  );
};

export default EnvironmentsPage;
