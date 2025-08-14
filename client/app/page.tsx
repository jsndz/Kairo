"use client";

import { useAuth } from "@/hooks/useAuth";
import { useDoc } from "@/hooks/useDoc";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

export default function Dashboard() {
  const { user, isAuthenticated, isLoading: authLoading, logout } = useAuth();
  const { docs, loading: docsLoading, error, fetchDocs, createDoc } = useDoc();
  const router = useRouter();

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push("/signup");
    }
  }, [authLoading, isAuthenticated, router]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchDocs();
    }
  }, [isAuthenticated, fetchDocs]);

  if (authLoading || docsLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p>Loading...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-500">{error}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      {/* Header */}
      <header className="flex items-center justify-between p-4 border-b border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800">
        <h1 className="text-lg font-semibold">Your Dashboard</h1>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            className="flex items-center gap-2"
            onClick={createDoc}
          >
            <Plus size={16} />
            Create Document
          </Button>
          <Button
            className="bg-blue-500 text-white hover:bg-blue-600"
            onClick={logout}
          >
            Logout
          </Button>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-4 space-y-6">
        {/* User Info */}
        <section className="p-4 bg-white dark:bg-gray-800 rounded shadow">
          <h2 className="text-md font-semibold mb-2">User Info</h2>
          {user ? (
            <ul className="space-y-1">
              <li>
                <strong>ID:</strong> {user.id}
              </li>
              <li>
                <strong>Name:</strong> {user.name}
              </li>
              <li>
                <strong>Email:</strong> {user.email}
              </li>
            </ul>
          ) : (
            <p>No user data available</p>
          )}
        </section>

        {/* User Documents */}
        <section className="p-4 bg-white dark:bg-gray-800 rounded shadow">
          <div className="flex items-center justify-between mb-2">
            <h2 className="text-md font-semibold">Your Documents</h2>
            <Button
              variant="secondary"
              size="sm"
              onClick={createDoc}
              className="flex items-center gap-1"
            >
              <Plus size={14} /> New
            </Button>
          </div>
          {docs && docs.length > 0 ? (
            <ul className="space-y-2">
              {docs.map((doc) => (
                <li
                  key={doc.id}
                  className="p-2 border border-gray-300 dark:border-gray-700 rounded hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
                  onClick={() => router.push(`/docs/${doc.id}`)}
                >
                  <strong>{doc.title || "Untitled Document"}</strong>
                  <div className="text-xs text-gray-500">
                    Updated: {new Date(doc.updated_at).toLocaleString()}
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <p>No documents found</p>
          )}
        </section>
      </main>
    </div>
  );
}
