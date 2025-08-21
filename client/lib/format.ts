function createMessage(type: number, payload: Uint8Array): Uint8Array {
  const message = new Uint8Array(1 + payload.length);
  message[0] = type;
  message.set(payload, 1);
  return message;
}

function parseMessage(message: Uint8Array): {
  type: number;
  payload: Uint8Array;
} {
  const type = message[0];
  const payload = message.slice(1);
  return { type, payload };
}
