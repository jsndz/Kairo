import axios from "axios";
import { Docs, CreateDocResponse, UpdateDocResponse } from "@/types/doc";
const API_BASE = "http://localhost:8080/api/v1/doc";

async function createDoc(userId: number): Promise<CreateDocResponse> {
  try {
    const res = await axios.post(
      `${API_BASE}/create`,
      { user_id: userId },
      { withCredentials: true }
    );
    return { success: true, doc: res.data.doc };
  } catch (err: any) {
    return {
      success: false,
      message:
        err.response?.data?.message ||
        err.response?.data?.error ||
        err.message ||
        "Failed to create document",
    };
  }
}

async function updateDoc(
  id: number,
  title: string,
  currentState: Uint8Array
): Promise<UpdateDocResponse> {
  try {
    const res = await axios.put(
      `${API_BASE}/update/${id}`,
      {
        title,
        current_state: currentState,
      },
      { withCredentials: true }
    );
    return { success: true, doc: res.data.doc };
  } catch (err: any) {
    return {
      success: false,
      message:
        err.response?.data?.message ||
        err.response?.data?.error ||
        err.message ||
        "Failed to update document",
    };
  }
}

async function updateName(id: number, title: string): Promise<string | null> {
  try {
    const res = await axios.put(
      `${API_BASE}/update/name/${id}`,
      {
        new_title: title,
      },
      { withCredentials: true }
    );

    return res.data.new_title;
  } catch (err: any) {
    return null;
  }
}

async function getDocMeta(id: number): Promise<Docs | null> {
  try {
    const res = await axios.get(`${API_BASE}/doc/${id}`, {
      headers: { Accept: "application/json" },
      withCredentials: true,
    });
    localStorage.setItem("ws_token", res.data.ws_token);

    return res.data.document;
  } catch {
    return null;
  }
}

async function getDocContent(id: number): Promise<Uint8Array | null> {
  try {
    const res = await axios.get(`${API_BASE}/doc/${id}`, {
      withCredentials: true,
      headers: { Accept: "application/octet-stream" },
      responseType: "arraybuffer",
    });

    return new Uint8Array(res.data);
  } catch {
    return null;
  }
}

async function getUserDocs(userId: number): Promise<Docs[]> {
  try {
    const res = await axios.get(`${API_BASE}/${userId}`, {
      withCredentials: true,
    });
    return res.data.docs || [];
  } catch {
    return [];
  }
}

async function AutoSave(id: number): Promise<Boolean> {
  try {
    const res = await axios.get(`${API_BASE}/save/${id}`, {
      withCredentials: true,
    });
    return res.data.success;
  } catch {
    return false;
  }
}

export const docService = {
  createDoc,
  updateDoc,
  getDocMeta,
  getDocContent,
  getUserDocs,
  updateName,
  AutoSave,
};
