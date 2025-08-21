"use client";

import { useParams } from "next/navigation";
import React, { useEffect, useRef, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";

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
  const docRef = useRef<Y.Doc | null>(null);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!docRef.current) {
    docRef.current = new Y.Doc();
  }
  const doc_id = parseInt(id!);

  const editor = useEditor({
    extensions: [
      Document,
      Paragraph,
      Text,
      Collaboration.configure({ document: docRef.current }),
    ],
    immediatelyRender: false,
  });

  useEffect(() => {
    if (!wsRef.current) {
      const token = localStorage.getItem("ws_token");
      const ws = new WebSocket(`ws://localhost:3004/ws`);
      wsRef.current = ws;

      ws.onopen = () => {
        ws.send(
          JSON.stringify({
            type: "join",
            payload: { token, doc_id },
          })
        );
      };

      ws.onmessage = (event) => {
        // [0, ...yjs_update_bytes] for document updates
        // [1, ...awareness_bytes] for awareness updates
        // [2, ...json_bytes] for auth/join
        const { type, payload } = parseMessage(event.data);
        switch (type) {
          case 0:
            Y.applyUpdate(docRef.current!, payload);
            break;
          case 1:
            break;
          // will be completed
          case 2:
            console.log(String(payload));
            break;
        }

        if (typeof event.data === "string") {
          const obj = JSON.parse(event.data);
          const values = Object.values(obj);
          const uint8 = new Uint8Array(values as number[]);
          Y.applyUpdate(docRef.current!, uint8);
        } else {
          Y.applyUpdate(docRef.current!, event.data as Uint8Array);
        }
      };

      ws.onclose = () => {
        wsRef.current = null;
      };
    }

    return () => {
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [doc_id]);

  useEffect(() => {
    if (CurrentState && CurrentState.length > 0) {
      Y.applyUpdate(docRef.current!, CurrentState);
    }

    const updateHandler = (update: Uint8Array) => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "update",
            payload: update,
          })
        );
      }

      if (onChangeState) {
        onChangeState(update);
      }
    };

    docRef.current?.on("update", updateHandler);
    return () => {
      docRef.current?.off("update", updateHandler);
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
