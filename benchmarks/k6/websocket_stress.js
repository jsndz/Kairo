import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '3s', target: 5 },   // Ramp-up
    { duration: '10s', target: 15 }, // Active collaborative sessions
    { duration: '2s', target: 0 },   // Clean disconnect
  ],
  thresholds: {
    ws_connecting: ['p(95)<100'], // Handshake within 100ms
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8081';
const WS_URL = BASE_URL.replace(/^http/, 'ws');

export default function () {
  // 1. Provision dedicated room
  const roomRes = http.post(`${BASE_URL}/api/rooms`, JSON.stringify({ name: `ws-room-${__VU}` }), {
    headers: { 'Content-Type': 'application/json' },
  });
  if (roomRes.status !== 201) return;

  const roomBody = JSON.parse(roomRes.body || '{}');
  const roomId = roomBody.roomId;
  const clientId = `k6_vu_${__VU}_${__ITER}`;
  const url = `${WS_URL}/api/projects/${roomId}/ws?clientId=${clientId}`;

  let currentRev = 0;
  let acksReceived = 0;
  let editsSent = 0;

  // 2. Connect WebSocket
  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      // Send initial presence heartbeat
      socket.send(JSON.stringify({
        type: 'presence',
        playhead: 120,
      }));
    });

    socket.on('message', (data) => {
      let msg;
      try {
        msg = JSON.parse(data);
      } catch (e) {
        return;
      }

      if (msg.type === 'init') {
        currentRev = msg.currentSeq || 0;

        // Send first timeline edit operation
        const reqId = `req_${__VU}_1`;
        editsSent++;
        socket.send(JSON.stringify({
          type: 'op_submit',
          reqId: reqId,
          expectedRevision: currentRev,
          op: {
            id: `edit_${__VU}_1`,
            type: 'rename',
            expectedRevision: currentRev,
            payload: { name: `Live Collab Session ${__VU}` },
          },
        }));
      } else if (msg.type === 'op_ack') {
        acksReceived++;
        currentRev = msg.seq;

        // Send second chained operation if under limit
        if (editsSent < 3) {
          const nextReq = `req_${__VU}_${editsSent + 1}`;
          editsSent++;
          socket.send(JSON.stringify({
            type: 'op_submit',
            reqId: nextReq,
            expectedRevision: currentRev,
            op: {
              id: `edit_${__VU}_${editsSent}`,
              type: 'rename',
              expectedRevision: currentRev,
              payload: { name: `Live Revision ${currentRev + 1}` },
            },
          }));
        } else {
          // Finished edit loop, close cleanly
          socket.setTimeout(() => {
            socket.close();
          }, 300);
        }
      }
    });

    socket.setTimeout(() => {
      socket.close();
    }, 2500);
  });

  check(res, {
    'websocket upgrade 101': (r) => r && r.status === 101,
  });

  sleep(0.1);
}
