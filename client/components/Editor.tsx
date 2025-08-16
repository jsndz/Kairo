"use client";

import { useParams } from "next/navigation";
import React, { useEffect, useRef, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";

const doc = new Y.Doc();

interface EditorPageProps {
  onChangeState?: (state: Uint8Array) => void;
  CurrentState: Uint8Array;
}

export default function EditorPage({
  onChangeState,
  CurrentState,
}: EditorPageProps) {
  const [isMounted, setIsMounted] = useState(false);
  const { id } = useParams<{ id: string }>();

  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  const decoded = new TextDecoder().decode(CurrentState || new Uint8Array());
  let parsed: any = null;
  try {
    parsed = decoded ? JSON.parse(decoded) : null;
  } catch (e) {
    console.warn("Invalid JSON in CurrentState:", e);
  }

  const doc_id = parseInt(id!);

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
    content: decoded,
    onUpdate: ({ editor }) => {
      const json = editor.getJSON();
      console.log("Editor updated:", json);

      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "update",
            payload: { doc_id, state: json },
          })
        );
      }

      if (onChangeState) {
        const encoded = new TextEncoder().encode(JSON.stringify(json));
        onChangeState(encoded);
      }
    },
  });

  useEffect(() => {
    if (!wsRef.current) {
      const token = localStorage.getItem("ws_token");
      const ws = new WebSocket(`ws://localhost:3004/ws`);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("Connected to server with token:", token);
        ws.send(
          JSON.stringify({
            type: "join",
            payload: { token: token, doc_id },
          })
        );
      };

      ws.onmessage = (event) => {
        console.log("Message from server:", event.data);

        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "update" && editor) {
            editor.commands.setContent(msg.payload.state);
          }
        } catch (e) {
          console.warn("Invalid WS message:", event.data);
        }
      };

      ws.onclose = () => {
        console.log("WebSocket closed, cleaning up");
        wsRef.current = null;
      };
    }

    return () => {
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [doc_id]);

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
