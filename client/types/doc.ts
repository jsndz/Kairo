export interface Docs {
  id: number;
  user_id: number;
  title: string;
  current_state: Uint8Array;
  created_at: string;
  updated_at: string;
}

export interface CreateDocResponse {
  success: boolean;
  doc?: Docs;
  message?: string;
}

export interface UpdateDocResponse {
  success: boolean;
  doc?: Docs;
  message?: string;
}
