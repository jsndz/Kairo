"use client";

import EditorPage from "@/components/Editor";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/useAuth";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const { logout, isAuthenticated, isLoading, user } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/signup");
    }
  }, [isLoading, isAuthenticated, router]);
  console.log("user", user);

  const handleSave = () => {
    // TODO: Implement save logic here
    console.log("Document saved!");
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      <header className="flex items-center justify-between p-4 border-b border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800">
        <h1 className="text-lg font-semibold">Kairo Editor</h1>
        <div className="flex gap-2">
          {/* Logout Button */}
          <Button
            className="flex items-center gap-2 bg-blue-500 text-white hover:bg-blue-600"
            onClick={logout}
          >
            Logout
          </Button>
          <Button onClick={handleSave}>Save</Button>
        </div>
      </header>

      {/* Editor */}
      <main className="flex-1 overflow-auto p-4">
        <EditorPage />
      </main>
    </div>
  );
}
