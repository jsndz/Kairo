export function createMessage(type: number, payload: Uint8Array): Uint8Array {
  const message = new Uint8Array(1 + payload.length);
  message[0] = type;
  message.set(payload, 1);
  return message;
}

export async function parseMessage(msg: Blob | ArrayBuffer): Promise<{
  type: number;
  payload: Uint8Array;
}> {
  const buffer =
    msg instanceof Blob ? await msg.arrayBuffer() : (msg as ArrayBuffer);

  const message = new Uint8Array(buffer);
  const type = message[0];
  const payload = message.slice(1);
  return { type, payload };
}
