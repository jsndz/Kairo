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

async function getDoc(id: number): Promise<Docs | null> {
  try {
    const res = await axios.get(`${API_BASE}/doc/${id}`, {
      withCredentials: true,
    });
    localStorage.setItem("ws_token", res.data.ws_token);
    return res.data.document;
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

export const docService = {
  createDoc,
  updateDoc,
  getDoc,
  getUserDocs,
};
