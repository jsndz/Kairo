"use client";

import React, { useState, useEffect } from "react";
import EditorPage from "@/components/Editor";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Save, Upload, Pencil } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useDoc } from "@/hooks/useDoc";
import { useParams } from "next/navigation";

const EditPage = () => {
  const { id } = useParams();
  const docId = Number(id);

  const { updateDoc, getDocById, changeTitle } = useDoc();
  const [title, setTitle] = useState("LOADING");
  const [editingTitle, setEditingTitle] = useState(false);
  const [currentState, setCurrentState] = useState<Uint8Array>(
    new Uint8Array()
  );

  useEffect(() => {
    const fetchData = async () => {
      if (!docId) return;
      const doc = await getDocById(docId);
      if (doc) {
        setTitle(doc.meta?.title || "");
        if (doc.content) {
          console.log("state", doc.content);
          setCurrentState(doc.content);
        }
      }
    };
    fetchData();
  }, [docId, getDocById]);

  const handleSave = async () => {
    if (!docId) return;
    await updateDoc(docId, title, currentState);
  };

  const handlePublish = () => {};

  const handleRename = () => {
    setEditingTitle(true);
  };

  const handleRenameSubmit = async () => {
    setEditingTitle(false);
    if (docId) {
      await changeTitle(docId, title);
    }
  };

  return (
    <main className="flex flex-col h-full bg-gray-50">
      {/* Top Toolbar */}
      <header className="flex items-center justify-between p-4 bg-white shadow-sm border-b">
        <div className="flex items-center gap-2">
          {editingTitle ? (
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onBlur={handleRenameSubmit}
              autoFocus
            />
          ) : (
            <>
              <h1 className="text-lg font-semibold">{title || "Untitled"}</h1>
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

        <div className="flex items-center gap-3">
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
      </header>

      {/* Editor Area */}
      <section className="flex-1 p-6 overflow-auto">
        <Card className="w-full h-full shadow-md">
          <CardContent className="p-4 h-full">
            <EditorPage
              CurrentState={currentState}
              onChangeState={setCurrentState}
            />
          </CardContent>
        </Card>
      </section>
    </main>
  );
};

export default EditPage;
