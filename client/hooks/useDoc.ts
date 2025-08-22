import { docService } from "@/services/doc";
import { Docs } from "@/types/doc";
import { useCallback, useState } from "react";
import { useAuth } from "./useAuth";

export function useDoc() {
  const [docs, setDocs] = useState<Docs[]>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuth();

  const fetchDocs = useCallback(async () => {
    if (!user?.id) return;
    setLoading(true);
    setError(null);
    try {
      const data = await docService.getUserDocs(user.id);
      setDocs(data);
    } catch (err: any) {
      setError(err.message || "Failed to fetch docs");
    } finally {
      setLoading(false);
    }
  }, [user?.id]);

  const getDocById = useCallback(async (id: number) => {
    setLoading(true);
    setError(null);
    try {
      const meta = await docService.getDocMeta(id);
      const content = await docService.getDocContent(id);
      console.log("D", { meta, content });

      return { meta, content };
    } catch (err: any) {
      setError(err.message || "Failed to fetch document");
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const changeTitle = useCallback(async (id: number, title: string) => {
    setLoading(true);
    setError(null);
    try {
      const data = await docService.updateName(id, title);

      return data;
    } catch (err: any) {
      setError(err.message || "Failed to change name");
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const createDoc = useCallback(async () => {
    if (!user?.id) return;
    setLoading(true);
    try {
      const res = await docService.createDoc(user.id);
      if (res.success && res.doc) {
        setDocs((prev) => (prev ? [...prev, res.doc!] : [res.doc!]));
      } else {
        throw new Error(res.message || "Failed to create document");
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [user?.id]);

  const updateDoc = useCallback(
    async (id: number, title: string, currentState: Uint8Array) => {
      setLoading(true);
      try {
        const res = await docService.updateDoc(id, title, currentState);
        if (res.success && res.doc) {
          setDocs((prev) =>
            (prev ?? []).map((doc) => (doc.id === id ? res.doc! : doc))
          );
        } else {
          throw new Error(res.message || "Failed to update document");
        }
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return {
    docs,
    loading,
    error,
    changeTitle,
    fetchDocs,
    createDoc,
    updateDoc,
    getDocById,
  };
}
