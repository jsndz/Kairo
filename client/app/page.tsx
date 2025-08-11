"use client";

import EditorPage from "@/components/Editor";
import MyEditor from "@/components/Editor";
import { Button } from "@/components/ui/button";
import { Rocket } from "lucide-react";

export default function Home() {
  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      {/* Toolbar */}
      <header className="flex items-center justify-between p-4 border-b border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800">
        <h1 className="text-lg font-semibold">Kairo Editor</h1>
        <Button>Save</Button>
      </header>

      {/* Editor */}
      <main className="flex-1 overflow-auto p-4">
        <EditorPage />
      </main>
    </div>
  );
}
