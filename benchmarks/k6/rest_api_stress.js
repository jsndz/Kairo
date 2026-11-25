import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 20 },   // Warm-up ramp
    { duration: '15s', target: 50 },  // Normal load
    { duration: '10s', target: 100 }, // Peak stress spike
    { duration: '5s', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<15', 'p(99)<30'], // 95% of requests under 15ms
    http_req_failed: ['rate<0.01'],              // Error rate under 1%
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8080';

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/api/health`);
  check(healthRes, {
    'health status is 200': (r) => r.status === 200,
  });

  // 2. Room provisioning
  const roomPayload = JSON.stringify({ name: `k6-room-${__VU}-${__ITER}` });
  const roomRes = http.post(`${BASE_URL}/api/rooms`, roomPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(roomRes, {
    'room created 201': (r) => r.status === 201,
  });

  const body = JSON.parse(roomRes.body || '{}');
  const projectId = body.roomId;

  if (projectId) {
    // 3. Project state retrieval
    const projRes = http.get(`${BASE_URL}/api/projects/${projectId}`);
    check(projRes, {
      'get project is 200': (r) => r.status === 200,
    });
  }

  sleep(0.05); // 50ms pacing between iterations
}
