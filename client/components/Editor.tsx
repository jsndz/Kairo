"use client";

import { useParams } from "next/navigation";
import React, { useEffect, useRef, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";
import { Toaster, toast } from "sonner";
import { createMessage, parseMessage } from "@/lib/format";

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
  const docRef = useRef<Y.Doc>(new Y.Doc());

  useEffect(() => {
    setIsMounted(true);
  }, []);

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
    if (wsRef.current) return;

    const token = localStorage.getItem("ws_token");
    const ws = new WebSocket(`ws://localhost:3004/ws`);
    wsRef.current = ws;

    ws.onopen = () => {
      const payload = new TextEncoder().encode(
        JSON.stringify({ token, doc_id })
      );
      ws.send(createMessage(2, payload));
    };

    ws.onmessage = async (event) => {
      const data =
        event.data instanceof Blob
          ? new Uint8Array(await event.data.arrayBuffer())
          : event.data;

      const { type, payload } = await parseMessage(data);

      switch (type) {
        case 0: // document update
          Y.applyUpdate(docRef.current, payload);
          break;
        case 1: // awareness update (TODO)
          break;
        case 2: // join message
          toast(String(payload));
          break;
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [doc_id]);

  useEffect(() => {
    const updateHandler = (update: Uint8Array) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(createMessage(0, update));
      }
      onChangeState?.(update);
    };

    if (CurrentState?.length > 0) {
      Y.applyUpdate(docRef.current, CurrentState);
    }

    docRef.current.on("update", updateHandler);

    return () => {
      docRef.current.off("update", updateHandler);
    };
  }, [doc_id, CurrentState, onChangeState]);

  if (!isMounted || !editor) return null;

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      <Toaster position="top-right" />

      <main className="flex-1 overflow-auto p-4">
        <EditorContent
          editor={editor}
          className="prose dark:prose-invert max-w-none h-full outline-none"
        />
      </main>
    </div>
  );
}
