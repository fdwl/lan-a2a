import WebSocket from "ws";
import { Bonjour } from "bonjour-service";
import { Message, newMsgID, sendMsg, readMsg, handshakeAsClient } from "./protocol.js";

const SERVICE_TYPE = "_lan-agent-bus._tcp";
const RELAY_SERVICE_TYPE = "_lan-agent-bus-relay._tcp";

interface Peer {
  id: string;
  addr: string;
  ws: WebSocket;
  lastSeen: number;
}

export type MessageHandler = (msg: Message, from: string) => void;

export class P2P {
  private peers = new Map<string, Peer>();
  private online = new Map<string, string>(); // peerID → addr
  private server: WebSocket.Server | null = null;
  private agentID: string;
  private port: number;
  private bonjour: Bonjour;
  private onMessage: MessageHandler | null = null;
  private onFileData: MessageHandler | null = null;
  private onRelay: ((addr: string) => void) | null = null;

  constructor(agentID: string, port: number) {
    this.agentID = agentID;
    this.port = port;
    this.bonjour = new Bonjour();
  }

  setMessageHandler(h: MessageHandler) { this.onMessage = h; }
  setFileDataHandler(h: MessageHandler) { this.onFileData = h; }
  setRelayHandler(h: (addr: string) => void) { this.onRelay = h; }

  start(): Promise<void> {
    return new Promise((resolve) => {
      this.server = new WebSocket.Server({ port: this.port }, () => {
        console.log(`[p2p] listening on :${this.port}`);
        this.startDiscovery();
        resolve();
      });
      this.server.on("connection", (ws) => this.handleIncoming(ws));
    });
  }

  stop() {
    this.bonjour.unpublishAll();
    this.bonjour.destroy();
    this.server?.close();
    for (const peer of this.peers.values()) {
      peer.ws.close();
    }
    this.peers.clear();
  }

  getOnlinePeers(): string[] {
    return Array.from(this.online.keys());
  }

  markOnline(peerID: string, addr: string) {
    this.online.set(peerID, addr);
  }

  // On-demand WebSocket connection
  async connectTo(peerID: string): Promise<void> {
    if (this.peers.has(peerID)) return;
    const addr = this.online.get(peerID);
    if (!addr) throw new Error(`peer ${peerID} not online`);

    const ws = new WebSocket(`ws://${addr}/ws`);
    await new Promise<void>((resolve, reject) => {
      ws.on("open", async () => {
        try {
          const ok = await handshakeAsClient(ws, this.agentID);
          if (!ok) { ws.close(); reject(new Error("handshake failed")); return; }
          const peer: Peer = { id: peerID, addr, ws, lastSeen: Date.now() };
          this.peers.set(peerID, peer);
          console.log(`[conn] opened to ${peerID}`);
          this.readLoop(peer);
          resolve();
        } catch (e) { ws.close(); reject(e); }
      });
      ws.on("error", reject);
    });
  }

  disconnectFrom(peerID: string) {
    const peer = this.peers.get(peerID);
    if (peer) {
      peer.ws.close();
      this.peers.delete(peerID);
      console.log(`[conn] closed to ${peerID}`);
    }
  }

  private async handleIncoming(ws: WebSocket) {
    try {
      const msg = await readMsg(ws);
      if (msg.type !== "register") { ws.close(); return; }
      sendMsg(ws, { type: "register_ok", id: newMsgID(), from: this.agentID, ts: Date.now() });
      const peer: Peer = { id: msg.from, addr: "", ws, lastSeen: Date.now() };
      this.peers.set(msg.from, peer);
      console.log(`[p2p] peer connected: ${msg.from}`);
      this.readLoop(peer);
    } catch { ws.close(); }
  }

  private readLoop(peer: Peer) {
    const loop = async () => {
      try {
        while (peer.ws.readyState === WebSocket.OPEN) {
          const msg = await readMsg(peer.ws);
          peer.lastSeen = Date.now();
          this.handleMsg(msg, peer);
        }
      } catch {
        this.peers.delete(peer.id);
        console.log(`[p2p] peer disconnected: ${peer.id}`);
      }
    };
    loop();
  }

  private handleMsg(msg: Message, peer: Peer) {
    switch (msg.type) {
      case "ping":
        sendMsg(peer.ws, { type: "pong", id: newMsgID(), from: this.agentID, ts: Date.now() });
        break;
      case "text": case "file_meta": case "file_done":
        this.onMessage?.(msg, peer.id);
        break;
      case "file_data":
        this.onFileData?.(msg, peer.id);
        break;
    }
  }

  private startDiscovery() {
    this.bonjour.publish({
      name: this.agentID, type: SERVICE_TYPE, port: this.port,
      txt: { id: this.agentID, version: "0.1.0" },
    });
    console.log(`[mdns] registered as ${this.agentID}`);

    this.bonjour.find({ type: SERVICE_TYPE }, (svc) => {
      if (svc.name === this.agentID) return;
      const peerID = (svc.txt as any)?.id || svc.name;
      const addr = svc.referer?.address || svc.addresses?.[0];
      if (addr) {
        this.markOnline(peerID, `${addr}:${svc.port}`);
        console.log(`[online] ${peerID} (lan)`);
      }
    });

    if (this.onRelay) {
      this.bonjour.find({ type: RELAY_SERVICE_TYPE }, (svc) => {
        const addr = svc.referer?.address || svc.addresses?.[0];
        if (addr) this.onRelay!(`${addr}:${svc.port}`);
      });
    }
  }
}
