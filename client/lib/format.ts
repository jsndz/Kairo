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

export function parseUint8Array(): string {
  const obj: Record<string, number> = {
    "0": 1, "1": 3, "2": 228, "3": 180, "4": 250, "5": 147, "6": 4,
    "7": 5, "8": 132, "9": 228, "10": 180, "11": 250, "12": 147, "13": 4,
    "14": 4, "15": 1, "16": 119, "17": 132, "18": 228, "19": 180, "20": 250,
    "21": 147, "22": 4, "23": 5, "24": 1, "25": 100, "26": 132, "27": 228,
    "28": 180, "29": 250, "30": 147, "31": 4, "32": 6, "33": 1, "34": 102, "35": 0
  };

  // Convert numeric-keyed object to an array of numbers
  const byteArray = new Uint8Array(
    Object.keys(obj)
      .sort((a, b) => Number(a) - Number(b)) // Ensure correct order
      .map(k => obj[k])
  );

  // Decode the Uint8Array to string
  const str = new TextDecoder().decode(byteArray);
console.log(str);

  return str;
}
