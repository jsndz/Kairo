import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 10 },
    { duration: '15s', target: 30 },
    { duration: '5s', target: 0 },
  ],
  thresholds: {
    ws_connecting: ['p(95)<100'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8080';
const WS_URL = BASE_URL.replace(/^http/, 'ws');

export default function () {
  // 1. Create room for this VU session
  const roomRes = http.post(`${BASE_URL}/api/rooms`, JSON.stringify({ name: `k6-ws-${__VU}` }), {
    headers: { 'Content-Type': 'application/json' },
  });
  if (roomRes.status !== 201) return;
  const roomId = JSON.parse(roomRes.body).roomId;

  const clientId = `k6_vu_${__VU}_${__ITER}`;
  const url = `${WS_URL}/api/projects/${roomId}/ws?clientId=${clientId}`;

  // 2. Connect WebSocket
  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      // Send presence update
      socket.send(JSON.stringify({
        type: 'presence',
        playhead: 150,
      }));
    });

    socket.on('message', (data) => {
      const msg = JSON.parse(data);
      check(msg, {
        'received message envelope': (m) => m && m.type !== undefined,
      });
    });

    socket.setTimeout(() => {
      socket.close();
    }, 2000);
  });

  check(res, { 'connected successfully': (r) => r && r.status === 101 });
  sleep(0.1);
}
