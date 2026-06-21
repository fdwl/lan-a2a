import WebSocket from "ws";

export type MsgType =
  | "register"
  | "register_ok"
  | "ping"
  | "pong"
  | "text"
  | "file_meta"
  | "file_data"
  | "file_done"
  | "query_online"
  | "online_list";

export interface Message {
  type: MsgType;
  id: string;
  from: string;
  channel_id?: string;
  content?: string;
  ts: number;
  filename?: string;
  file_size?: number;
  checksum?: string;
  chunk_idx?: number;
  total_chunks?: number;
  data?: Buffer;
}

export function newMsgID(): string {
  return `${Date.now()}-${Math.random().toString(16).slice(2, 10)}`;
}

export function sendMsg(ws: WebSocket, msg: Message): void {
  ws.send(JSON.stringify(msg));
}

export function readMsg(ws: WebSocket): Promise<Message> {
  return new Promise((resolve, reject) => {
    const handler = (data: WebSocket.Data) => {
      ws.removeListener("message", handler);
      ws.removeListener("error", reject);
      try {
        resolve(JSON.parse(data.toString()) as Message);
      } catch (e) {
        reject(e);
      }
    };
    ws.on("message", handler);
    ws.on("error", reject);
  });
}

export async function handshakeAsClient(ws: WebSocket, agentID: string): Promise<boolean> {
  sendMsg(ws, { type: "register", id: newMsgID(), from: agentID, ts: Date.now() });
  const resp = await readMsg(ws);
  return resp.type === "register_ok";
}

export async function handshakeAsServer(ws: WebSocket, serverID: string): Promise<string | null> {
  const msg = await readMsg(ws);
  if (msg.type !== "register") return null;
  sendMsg(ws, { type: "register_ok", id: newMsgID(), from: serverID, ts: Date.now() });
  return msg.from;
}
