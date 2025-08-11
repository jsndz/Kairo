"use client";

import React, { useEffect, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import * as Y from "yjs";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";

import Collaboration from "@tiptap/extension-collaboration";
const doc = new Y.Doc();

export default function EditorPage() {
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  const editor = useEditor({
    extensions: [
      Document,
      Paragraph,
      Text,
      Collaboration.configure({
        document: doc,
      }),
    ],
  });
  useEffect(() => {
    // Example: sending JWT token as query parameter
    const token = localStorage.getItem("jwt") || "test-token";
    const provider = new WebSocket("ws://localhost:3004/ws");

    provider.onopen = () => {
      console.log("WebSocket connection established");
      provider.send(JSON.stringify({ documentId: "my-document-id" }));
    };

    provider.onopen = (e: Event) => {
      console.log("WebSocket connection established");
      provider.send(
        JSON.stringify({
          documentId: "my-document-id",
          token: token as string,
        })
      );
    };
  }, []);
  if (!isMounted || !editor) return null;

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      {/* Editor */}
      <main className="flex-1 overflow-auto p-4">
        <EditorContent
          editor={editor}
          className="prose dark:prose-invert max-w-none h-full outline-none"
        />
      </main>
    </div>
  );
}
