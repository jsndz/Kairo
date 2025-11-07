"use client";

import { useAuth } from "@/hooks/useAuth";
import { useDoc } from "@/hooks/useDoc";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Plus,
  FileText,
  User,
  Mail,
  Calendar,
  LogOut,
  Sparkles,
  FolderOpen,
  Clock,
  Loader2,
} from "lucide-react";

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
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-slate-50 via-white to-slate-50">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-8 h-8 animate-spin text-slate-600" />
          <p className="text-slate-600 font-medium">Loading your workspace...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-slate-50 via-white to-slate-50">
        <Card className="max-w-md border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-800">
          <CardContent className="pt-6">
            <p className="text-red-600 dark:text-red-400 font-medium">{error}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInHours = Math.floor(
      (now.getTime() - date.getTime()) / (1000 * 60 * 60)
    );

    if (diffInHours < 1) return "Just now";
    if (diffInHours < 24) return `${diffInHours}h ago`;
    if (diffInHours < 48) return "Yesterday";
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
    });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-50 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950">
      {/* Header */}
      <header className="sticky top-0 z-50 backdrop-blur-xl bg-white/70 dark:bg-slate-900/70 border-b border-slate-200/50 dark:border-slate-800/50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-slate-900 to-slate-700 dark:from-slate-100 dark:to-slate-300 flex items-center justify-center shadow-lg">
                <Sparkles className="w-5 h-5 text-white dark:text-slate-900" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-slate-900 dark:text-white">
                  Dashboard
                </h1>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Welcome back, {user?.name || "User"}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Button
                variant="outline"
                className="flex items-center gap-2 border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800"
                onClick={createDoc}
              >
                <Plus className="w-4 h-4" />
                Create Document
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                onClick={logout}
                title="Logout"
              >
                <LogOut className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid gap-6 md:grid-cols-3">
          {/* User Info Card */}
          <Card className="md:col-span-1 border-slate-200 dark:border-slate-800 shadow-sm hover:shadow-md transition-shadow">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-white">
                <User className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                Profile
              </CardTitle>
              <CardDescription>Your account information</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {user ? (
                <>
                  <div className="flex items-start gap-3">
                    <div className="w-12 h-12 rounded-full bg-gradient-to-br from-slate-900 to-slate-700 dark:from-slate-100 dark:to-slate-300 flex items-center justify-center text-white dark:text-slate-900 font-bold text-lg shadow-md">
                      {user.name
                        ?.split(" ")
                        .map((n) => n[0])
                        .join("")
                        .toUpperCase() || "U"}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold text-slate-900 dark:text-white truncate">
                        {user.name}
                      </p>
                      <p className="text-sm text-slate-500 dark:text-slate-400 truncate flex items-center gap-1">
                        <Mail className="w-3 h-3" />
                        {user.email}
                      </p>
                    </div>
                  </div>
                  <div className="pt-4 border-t border-slate-200 dark:border-slate-800">
                    <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                      <span className="font-mono">ID:</span>
                      <span className="font-mono">{user.id}</span>
                    </div>
                  </div>
                </>
              ) : (
                <p className="text-slate-500 dark:text-slate-400">
                  No user data available
                </p>
              )}
            </CardContent>
          </Card>

          {/* Documents Section */}
          <Card className="md:col-span-2 border-slate-200 dark:border-slate-800 shadow-sm">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-white">
                    <FolderOpen className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                    Your Documents
                  </CardTitle>
                  <CardDescription>
                    {docs && docs.length > 0
                      ? `${docs.length} document${docs.length !== 1 ? "s" : ""}`
                      : "No documents yet"}
                  </CardDescription>
                </div>
                <Button
                  size="sm"
                  className="flex items-center gap-2"
                  onClick={createDoc}
                >
                  <Plus className="w-4 h-4" />
                  New
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {docs && docs.length > 0 ? (
                <div className="space-y-3">
                  {docs.map((doc) => (
                    <div
                      key={doc.id}
                      className="group p-4 rounded-lg border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-900/50 cursor-pointer transition-all duration-200"
                      onClick={() => router.push(`/docs/${doc.id}`)}
                    >
                      <div className="flex items-start gap-3">
                        <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 dark:from-blue-600 dark:to-blue-700 flex items-center justify-center flex-shrink-0 shadow-sm group-hover:shadow-md transition-shadow">
                          <FileText className="w-5 h-5 text-white" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <h3 className="font-semibold text-slate-900 dark:text-white mb-1 truncate group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                            {doc.title || "Untitled Document"}
                          </h3>
                          <div className="flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400">
                            <span className="flex items-center gap-1">
                              <Clock className="w-3 h-3" />
                              {formatDate(doc.updated_at)}
                            </span>
                            <span className="flex items-center gap-1">
                              <Calendar className="w-3 h-3" />
                              {new Date(doc.updated_at).toLocaleDateString(
                                "en-US",
                                {
                                  month: "short",
                                  day: "numeric",
                                }
                              )}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <div className="w-16 h-16 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center mb-4">
                    <FileText className="w-8 h-8 text-slate-400 dark:text-slate-600" />
                  </div>
                  <h3 className="font-semibold text-slate-900 dark:text-white mb-2">
                    No documents yet
                  </h3>
                  <p className="text-sm text-slate-500 dark:text-slate-400 mb-6 max-w-sm">
                    Get started by creating your first document. Click the button
                    above to create one.
                  </p>
                  <Button onClick={createDoc} className="flex items-center gap-2">
                    <Plus className="w-4 h-4" />
                    Create Your First Document
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
