"use client";
import { useParams } from "next/navigation";
import React, { useEffect, useRef, useState, useCallback } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import Paragraph from "@tiptap/extension-paragraph";
import Document from "@tiptap/extension-document";
import Text from "@tiptap/extension-text";
import Collaboration from "@tiptap/extension-collaboration";
import * as Y from "yjs";
import { Toaster, toast } from "sonner";
import { createMessage, parseMessage } from "@/lib/format";
import { useDoc } from "@/hooks/useDoc";

interface EditorPageProps {
  onChangeState?: (state: Uint8Array) => void;
  currentState?: Uint8Array;
  initialState: Uint8Array;
  docRef?: React.RefObject<Y.Doc | null>;
}

export default function EditorPage({
  onChangeState,
  currentState,
  initialState,
  docRef: externalDocRef,
}: EditorPageProps) {
  const [isMounted, setIsMounted] = useState(false);
  const [isInitialized, setIsInitialized] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<
    "connecting" | "connected" | "disconnected"
  >("disconnected");
  const { getDocById } = useDoc();
  const { id } = useParams<{ id: string }>();
  const wsRef = useRef<WebSocket | null>(null);
  const internalDocRef = useRef<Y.Doc | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const appliedStatesRef = useRef<Set<string>>(new Set());

  const docRef = externalDocRef || internalDocRef;

  const docId = React.useMemo(() => {
    if (!id) return null;
    const parsed = parseInt(id, 10);
    return isNaN(parsed) ? null : parsed;
  }, [id]);

  useEffect(() => {
    if (!docRef.current) {
      docRef.current = new Y.Doc();
    }
  }, [docRef]);

  const editor = useEditor({
    extensions: [
      Document,
      Paragraph,
      Text,
      Collaboration.configure({
        document: docRef.current || new Y.Doc(),
      }),
    ],
    immediatelyRender: false,
  });

  const createUpdateHash = useCallback((update: Uint8Array): string => {
    return Array.from(update).join(",");
  }, []);

  useEffect(() => {
    if (!docRef.current || !docId) return;

    const fetchInitial = async () => {
      try {
        const res = await getDocById(docId);
        console.log(res?.content);
      } catch (error) {
        console.error("Failed to fetch or load initial state:", error);
        toast.error("Failed to fetch initial document");
      } finally {
        if (!isInitialized) {
          setIsInitialized(true);
        }
      }
    };

    fetchInitial();
  }, [docId, docRef, isInitialized, createUpdateHash, getDocById]);

  useEffect(() => {
    if (
      currentState &&
      currentState.length > 0 &&
      docRef.current &&
      isInitialized
    ) {
      try {
        const hash = createUpdateHash(currentState);
        if (!appliedStatesRef.current.has(hash)) {
          Y.applyUpdate(docRef.current, currentState);
          appliedStatesRef.current.add(hash);
        }
      } catch (error) {
        console.error("Failed to apply current state:", error);
        toast.error("Failed to apply document update");
      }
    }
  }, [currentState, docRef, isInitialized, createUpdateHash]);

  const connectWebSocket = useCallback(() => {
    if (!docId || wsRef.current?.readyState === WebSocket.OPEN) return;

    try {
      setConnectionStatus("connecting");

      const token =
        typeof window !== "undefined" ? localStorage.getItem("ws_token") : null;
      console.log(token);

      const ws = new WebSocket(`ws://localhost:3004/ws`);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("WebSocket connected");
        setConnectionStatus("connected");

        try {
          const payload = new TextEncoder().encode(
            JSON.stringify({ token, doc_id: docId })
          );
          ws.send(createMessage(2, payload));
        } catch (error) {
          toast.error("Failed to authenticate with server");
        }
      };

      ws.onmessage = async (event) => {
        try {
          const { type, payload } = await parseMessage(event.data);

          switch (type) {
            case 0:
              if (docRef.current && payload) {
                const hash = createUpdateHash(payload);
                if (!appliedStatesRef.current.has(hash)) {
                  Y.applyUpdate(docRef.current, payload);
                  appliedStatesRef.current.add(hash);
                }
              }
              break;
            case 1:
              break;
            case 2:
              if (payload) {
                const message = new TextDecoder().decode(payload);
                toast.info(message);
              }
              break;
            default:
              console.warn("Unknown message type:", type);
          }
        } catch (error) {
          console.error("Failed to process WebSocket message:", error);
          toast.error("Failed to process server message");
        }
      };

      ws.onclose = (event) => {
        setConnectionStatus("disconnected");
        wsRef.current = null;

        if (event.code !== 1000 && docId) {
          reconnectTimeoutRef.current = setTimeout(() => {
            connectWebSocket();
          }, 3000);
        }
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
        toast.error("Connection error occurred");
        setConnectionStatus("disconnected");
      };
    } catch (error) {
      console.error("Failed to create WebSocket connection:", error);
      setConnectionStatus("disconnected");
      toast.error("Failed to connect to server");
    }
  }, [docId, docRef, createUpdateHash]);

  useEffect(() => {
    if (isMounted && docId && isInitialized) {
      connectWebSocket();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close(1000, "Component unmounting");
        wsRef.current = null;
      }
    };
  }, [connectWebSocket, isMounted, docId, isInitialized]);

  useEffect(() => {
    if (!docRef.current) return;

    const updateHandler = (update: Uint8Array, origin: any) => {
      if (origin === "remote") return;

      const hash = createUpdateHash(update);
      appliedStatesRef.current.add(hash);

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        try {
          wsRef.current.send(createMessage(0, update));
        } catch (error) {
          console.error("Failed to send update to server:", error);
          toast.error("Failed to sync changes");
        }
      }

      if (onChangeState) {
        onChangeState(update);
      }
    };

    docRef.current.on("update", updateHandler);

    return () => {
      docRef.current?.off("update", updateHandler);
    };
  }, [docRef, onChangeState, createUpdateHash]);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!docId) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <h2 className="text-xl font-semibold text-red-600 dark:text-red-400 mb-2">
            Invalid Document ID
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            Please check the URL and try again.
          </p>
        </div>
      </div>
    );
  }

  if (!isMounted) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading editor...</p>
        </div>
      </div>
    );
  }

  if (!editor || !isInitialized) {
    return (
      <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
        <Toaster position="top-right" />

        <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <h1 className="text-lg font-semibold">Document Editor</h1>
            <div className="flex items-center space-x-2">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-500" />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Initializing...
              </span>
            </div>
          </div>
        </div>

        <main className="flex-1 overflow-auto p-4 flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
            <p className="text-gray-600 dark:text-gray-400">
              {!editor ? "Loading editor..." : "Initializing document..."}
            </p>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white">
      <Toaster position="top-right" />

      <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">Document Editor</h1>
          <div className="flex items-center space-x-2">
            <div
              className={`w-2 h-2 rounded-full ${
                connectionStatus === "connected"
                  ? "bg-green-500"
                  : connectionStatus === "connecting"
                  ? "bg-yellow-500"
                  : "bg-red-500"
              }`}
            />
            <span className="text-sm text-gray-600 dark:text-gray-400 capitalize">
              {connectionStatus}
            </span>
            {connectionStatus === "disconnected" && (
              <button
                onClick={connectWebSocket}
                className="text-xs px-2 py-1 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors"
              >
                Reconnect
              </button>
            )}
          </div>
        </div>
      </div>

      <main className="flex-1 overflow-auto p-4">
        <EditorContent
          editor={editor}
          className="prose dark:prose-invert max-w-none h-full outline-none focus:outline-none"
        />
      </main>
    </div>
  );
}
