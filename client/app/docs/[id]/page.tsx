"use client";

import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Save, Upload, Pencil } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useDoc } from "@/hooks/useDoc";
import { useParams } from "next/navigation";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";
import { Toaster, toast } from "sonner";
import { createMessage, parseMessage } from "@/lib/format";
import { Docs } from "@/types/doc";
interface DocumentMeta {
  id: number;
  title: string;
  createdAt: string;
  updatedAt: string;
  user_id: number;
}
const EditPage = () => {
  const isFirstEdit = useRef(true);
  const { id } = useParams<{ id: string }>();
  const { updateDoc, getDocById, changeTitle, AutoSave } = useDoc();
  const [title, setTitle] = useState("LOADING");
  const [editingTitle, setEditingTitle] = useState(false);
  const [currentState, setCurrentState] = useState<Uint8Array>(
    new Uint8Array()
  );
  const [isMounted, setIsMounted] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const docRef = useRef<Y.Doc | null>(null);
  const timeoutRef = useRef<number | null>(null);
  const [documentMeta, setDocumentMeta] = useState<Docs | null>(null);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!docRef.current) {
    docRef.current = new Y.Doc();
  }
  const doc_id = React.useMemo(() => {
    if (!id) return null;
    const parsed = parseInt(id, 10);
    return isNaN(parsed) ? null : parsed;
  }, [id]);
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
    const fetchData = async () => {
      const res = await getDocById(doc_id!);
      if (res?.meta) {
        setDocumentMeta(res.meta);
        setTitle(res.meta.title);
      }
      if (res?.content && res.content.byteLength > 0 && docRef.current) {
        console.log(res.content);

        Y.applyUpdate(docRef.current, res.content);
      }
    };
    fetchData();
  }, [doc_id, getDocById]);

  const handleSave = async () => {
    if (!doc_id) return;
    await updateDoc(doc_id, title, currentState);
  };

  const handlePublish = () => {};

  const handleRename = () => {
    setEditingTitle(true);
  };

  const handleRenameSubmit = async () => {
    setEditingTitle(false);
    if (doc_id) {
      await changeTitle(doc_id, title);
    }
  };

  useEffect(() => {
    if (!wsRef.current) {
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
        const { type, payload } = await parseMessage(event.data);

        switch (type) {
          case 0:
            Y.applyUpdate(docRef.current!, payload);
            break;
          case 1:
            break;
          case 2:
            toast(String(payload));
            break;
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
    if (currentState && currentState.length > 0) {
      Y.applyUpdate(docRef.current!, currentState);
    }
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
    }
    timeoutRef.current = window.setTimeout(() => {
      console.log("AutoSaving after pause:", doc_id);
      AutoSave(doc_id!);
    }, 2000);
    const updateHandler = (update: Uint8Array) => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(createMessage(0, update));
      }
      setCurrentState(update);
    };

    docRef.current?.on("update", updateHandler);
    return () => {
      docRef.current?.off("update", updateHandler);
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [doc_id]);

  if (!isMounted || !editor) return null;

  return (
    <main className="flex flex-col h-full bg-gray-50">
      <Toaster position="top-right" />

      {/* Top Toolbar */}
      <header className="flex flex-col p-4 bg-white shadow-sm border-b">
        <div className="flex items-center justify-between">
          <div className="flex flex-col">
            <div className="flex items-center gap-2">
              {editingTitle ? (
                <Input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  onBlur={handleRenameSubmit}
                  autoFocus
                  className="text-lg font-semibold w-64"
                />
              ) : (
                <>
                  <h1 className="text-lg font-semibold">
                    {title || documentMeta?.title || "Untitled"}
                  </h1>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleRename}
                    className="text-gray-500"
                  >
                    <Pencil size={16} />
                  </Button>
                </>
              )}
            </div>

            {/* Metadata Display */}
            {documentMeta && (
              <div className="text-xs text-gray-500 mt-1 flex flex-wrap gap-4">
                <span>ID: {documentMeta.id}</span>
                <span>User: {documentMeta.user_id}</span>
                <span>
                  Created: {new Date(documentMeta.created_at).toLocaleString()}
                </span>
                <span>
                  Updated: {new Date(documentMeta.updated_at).toLocaleString()}
                </span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-3 mt-2 sm:mt-0">
            <Button
              variant="outline"
              size="sm"
              onClick={handleSave}
              className="flex items-center gap-2"
            >
              <Save size={16} /> Save
            </Button>
            <Button
              size="sm"
              onClick={handlePublish}
              className="flex items-center gap-2"
            >
              <Upload size={16} /> Publish
            </Button>
          </div>
        </div>
      </header>

      {/* Editor Area */}
      <section className="flex-1 p-6 overflow-auto">
        <Card className="w-full h-full shadow-md">
          <CardContent className="p-4 h-full">
            <EditorContent
              editor={editor}
              className="prose dark:prose-invert max-w-none h-full outline-none"
            />
          </CardContent>
        </Card>
      </section>
    </main>
  );
};

export default EditPage;
