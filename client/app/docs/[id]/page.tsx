"use client";

import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Save, Upload, Pencil, Loader2 } from "lucide-react";
import { ThemeToggle } from "@/components/theme-toggle";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

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

  const [summary, setSummary] = useState<string | null>(null);
  const [showSummaryModal, setShowSummaryModal] = useState(false);
  const [loadingSummary, setLoadingSummary] = useState(false);

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

  const handleSummarize = () => {
    setShowSummaryModal(true);
    setLoadingSummary(true);
    setSummary(null);
    wsRef.current?.send(createMessage(3, new Uint8Array()));
  };

  useEffect(() => {
    const fetchData = async () => {
      const res = await getDocById(doc_id!);
      if (res?.meta) {
        setDocumentMeta(res.meta);
        setTitle(res.meta.title);
      }
      if (res?.content && res.content.byteLength > 0 && docRef.current) {
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
        console.log(type, payload);
        
        switch (type) {
          case 0:
            Y.applyUpdate(docRef.current!, payload);
            break;
          case 1:
            break;
          case 2:
            toast(new TextDecoder().decode(payload));
            break;
          case 3:
            try {
              const decoded = new TextDecoder().decode(payload);
              setSummary(decoded);
            } catch (err) {
              console.error("Failed to decode summary:", err);
              setSummary("Failed to load summary.");
            } finally {
              setLoadingSummary(false);
            }
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
    <main className="flex flex-col h-full bg-gradient-to-br from-slate-50 via-gray-50 to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950">
      <Toaster position="top-right" />

      {/* Top Toolbar */}
      <header className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-md border-b border-gray-200/50 dark:border-slate-800/50 shadow-sm sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-5">
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            {/* Title Section */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-3 mb-2">
                {editingTitle ? (
                  <Input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    onBlur={handleRenameSubmit}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        handleRenameSubmit();
                      }
                      if (e.key === "Escape") {
                        setEditingTitle(false);
                      }
                    }}
                    autoFocus
                    className="text-xl font-bold w-full max-w-md border-2 border-blue-500 focus:ring-2 focus:ring-blue-500/20 rounded-lg px-4 py-2"
                  />
                ) : (
                  <>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white truncate">
                      {title || documentMeta?.title || "Untitled Document"}
                    </h1>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleRename}
                      className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                    >
                      <Pencil size={18} />
                    </Button>
                  </>
                )}
              </div>

              {/* Metadata */}
              {documentMeta && (
                <div className="flex flex-wrap items-center gap-x-6 gap-y-1 text-xs text-gray-500 dark:text-gray-400 mt-2">
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-gray-100/60 dark:bg-slate-800/60 text-gray-600 dark:text-gray-300 font-medium">
                    <span className="font-semibold">ID:</span> {documentMeta.id}
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-gray-100/60 dark:bg-slate-800/60 text-gray-600 dark:text-gray-300 font-medium">
                    <span className="font-semibold">User:</span> {documentMeta.user_id}
                  </span>
                  <span className="text-gray-500 dark:text-gray-400">
                    Created {new Date(documentMeta.created_at).toLocaleDateString("en-US", { 
                      month: "short", 
                      day: "numeric", 
                      year: "numeric",
                      hour: "2-digit",
                      minute: "2-digit"
                    })}
                  </span>
                  <span className="text-gray-500 dark:text-gray-400">
                    Updated {new Date(documentMeta.updated_at).toLocaleDateString("en-US", { 
                      month: "short", 
                      day: "numeric", 
                      year: "numeric",
                      hour: "2-digit",
                      minute: "2-digit"
                    })}
                  </span>
                </div>
              )}
            </div>

            {/* Action Buttons */}
            <div className="flex items-center gap-2.5 flex-shrink-0">
              <ThemeToggle />
              <Button
                variant="secondary"
                size="default"
                onClick={handleSummarize}
                className="flex items-center gap-2 bg-gradient-to-r from-blue-500 via-blue-600 to-indigo-600 text-white hover:from-blue-600 hover:via-blue-700 hover:to-indigo-700 shadow-md hover:shadow-lg transition-all duration-200 px-5 py-2.5 rounded-lg font-medium"
              >
                <Pencil size={18} className="opacity-90" /> 
                Summarize
              </Button>
              <Button
                variant="outline"
                size="default"
                onClick={handleSave}
                className="flex items-center gap-2 border-2 hover:bg-gray-50 dark:hover:bg-slate-800 hover:border-gray-300 dark:hover:border-slate-700 transition-all duration-200 px-5 py-2.5 rounded-lg font-medium shadow-sm"
              >
                <Save size={18} /> 
                Save
              </Button>
              <Button
                size="default"
                onClick={handlePublish}
                className="flex items-center gap-2 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900 shadow-md hover:shadow-lg transition-all duration-200 px-5 py-2.5 rounded-lg font-medium"
              >
                <Upload size={18} /> 
                Publish
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Editor */}
      <section className="flex-1 overflow-auto">
        <div className="max-w-5xl mx-auto px-6 py-8">
          <Card className="w-full min-h-[calc(100vh-250px)] shadow-xl border-0 bg-white/90 dark:bg-slate-900/90 backdrop-blur-sm">
            <CardContent className="p-8 md:p-12 lg:p-16">
              <div className="max-w-3xl mx-auto">
                <EditorContent
                  editor={editor}
                  className="prose prose-lg dark:prose-invert max-w-none focus:outline-none prose-headings:font-bold prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-p:leading-relaxed prose-h1:text-4xl prose-h2:text-3xl prose-h3:text-2xl min-h-[400px]"
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Summary Modal */}
      <Dialog open={showSummaryModal} onOpenChange={setShowSummaryModal}>
        <DialogContent className="max-w-2xl bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border-0 dark:border-slate-800 p-0 overflow-hidden">
          <div className="bg-gradient-to-r from-blue-500 via-blue-600 to-indigo-600 px-6 py-5">
            <DialogHeader className="space-y-2">
              <DialogTitle className="text-2xl font-bold text-white flex items-center gap-2">
                <div className="p-2 bg-white/20 rounded-lg backdrop-blur-sm">
                  <Pencil size={20} className="text-white" />
                </div>
                AI Summary
              </DialogTitle>
              <DialogDescription className="text-blue-100 text-base">
                {loadingSummary
                  ? "AI is analyzing your document and generating a comprehensive summary..."
                  : "Here's the AI-generated summary of your document content."}
              </DialogDescription>
            </DialogHeader>
          </div>

          <div className="px-6 py-6">
            <div className="min-h-[200px] max-h-[400px] overflow-y-auto p-6 bg-gradient-to-br from-gray-50 to-gray-100/50 dark:from-slate-800 dark:to-slate-900/50 rounded-xl border border-gray-200 dark:border-slate-700">
              {loadingSummary ? (
                <div className="flex flex-col items-center justify-center gap-4 min-h-[200px] text-gray-600 dark:text-gray-400">
                  <Loader2 className="animate-spin w-8 h-8 text-blue-600 dark:text-blue-400" />
                  <p className="font-medium">Generating summary...</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">This may take a few moments</p>
                </div>
              ) : (
                <div className="whitespace-pre-wrap text-gray-800 dark:text-gray-200 leading-relaxed text-base">
                  {summary || (
                    <div className="text-center text-gray-500 dark:text-gray-400 py-8">
                      <p className="font-medium">No summary available.</p>
                      <p className="text-sm mt-2">Try generating a new summary.</p>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          <div className="px-6 py-4 bg-gray-50 dark:bg-slate-800 border-t border-gray-200 dark:border-slate-700 flex justify-end gap-3">
            <Button 
              variant="outline" 
              onClick={() => setShowSummaryModal(false)}
              className="px-6 rounded-lg font-medium"
            >
              Close
            </Button>
            {!loadingSummary && summary && (
              <Button 
                onClick={() => {
                  navigator.clipboard.writeText(summary);
                  toast.success("Summary copied to clipboard!");
                }}
                className="px-6 rounded-lg font-medium bg-blue-600 hover:bg-blue-700 text-white"
              >
                Copy Summary
              </Button>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </main>
  );
};

export default EditPage;
