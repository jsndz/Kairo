"use client";

import { useParams } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";

const doc = new Y.Doc();

interface EditorPageProps {
  onChangeState?: (state: Uint8Array) => void;
}

export default function EditorPage({ onChangeState }: EditorPageProps) {
  const [isMounted, setIsMounted] = useState(false);
  const { id } = useParams<{ id: string }>();
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
    immediatelyRender: false,
  });
  useEffect(() => {
    const ws = new WebSocket("ws://localhost:3004/ws");
    ws.onopen = () => {
      const token = document.cookie
        .split(";")
        .find((row) => row.startsWith("token="))
        ?.split("=")[1];
      console.log(id);
      console.log("token", token);

      ws.send(
        JSON.stringify({
          type: "join",
          token: token,
          doc_id: id,
        })
      );
    };
  });

  if (!isMounted || !editor) return null;

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      <main className="flex-1 overflow-auto p-4">
        <EditorContent
          editor={editor}
          className="prose dark:prose-invert max-w-none h-full outline-none"
        />
      </main>
    </div>
  );
}
