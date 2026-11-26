import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '3s', target: 5 },   // Ramp-up
    { duration: '8s', target: 15 },  // Concurrent render requests
    { duration: '2s', target: 0 },   // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<40'],
    http_req_failed: ['rate<0.02'],
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8081';

export default function () {
  // 1. Provision room
  const roomRes = http.post(`${BASE_URL}/api/rooms`, JSON.stringify({ name: `export-room-${__VU}` }), {
    headers: { 'Content-Type': 'application/json' },
  });
  if (roomRes.status !== 201) return;

  const roomBody = JSON.parse(roomRes.body || '{}');
  const projectId = roomBody.roomId;
  if (!projectId) return;

  // 2. Register media asset
  const assetRes = http.post(`${BASE_URL}/api/projects/${projectId}/assets`, JSON.stringify({
    kind: 'video',
    name: 'source.mp4',
    mime: 'video/mp4',
    bytes: 1048576,
    duration: 10,
    frames: 300,
    width: 1920,
    height: 1080,
  }), { headers: { 'Content-Type': 'application/json' } });
  if (assetRes.status !== 201) return;

  const asset = JSON.parse(assetRes.body || '{}');
  const assetId = asset.id;

  // 3. Begin & Complete upload
  const uploadRes = http.post(`${BASE_URL}/api/projects/${projectId}/assets/${assetId}/upload`);
  const uploadBody = JSON.parse(uploadRes.body || '{}');
  const attempt = uploadBody.asset ? uploadBody.asset.attempt : null;

  if (attempt) {
    http.post(`${BASE_URL}/api/projects/${projectId}/assets/${assetId}/complete`, JSON.stringify({ attempt: attempt }), {
      headers: { 'Content-Type': 'application/json' },
    });
  }

  // 4. Add clip to track
  const clipEditRes = http.post(`${BASE_URL}/api/projects/${projectId}/operations`, JSON.stringify({
    id: `clip_op_${__VU}_${__ITER}`,
    type: 'add',
    expectedRevision: 0,
    payload: {
      clip: {
        id: `clip_${__VU}_${__ITER}`,
        trackId: 'video-1',
        mediaId: assetId,
        at: 0,
        offset: 0,
        length: 300,
        volume: 1.0,
      },
    },
  }), { headers: { 'Content-Type': 'application/json' } });
  if (clipEditRes.status !== 200) return;

  const updatedProj = JSON.parse(clipEditRes.body || '{}');
  const revision = updatedProj.revision || 1;

  // 5. Dispatch video export render job
  const exportPayload = JSON.stringify({
    projectId: projectId,
    requestId: `export_req_${__VU}_${__ITER}`,
    revision: revision,
    format: 'mp4',
    filename: `output_${__VU}.mp4`,
    resolution: '1080p',
    frameRate: 30,
    quality: 'standard',
  });

  const exportRes = http.post(`${BASE_URL}/api/projects/${projectId}/exports`, exportPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const isAccepted = check(exportRes, {
    'export job submitted (200 or 202)': (r) => r.status === 200 || r.status === 202,
  });

  if (isAccepted) {
    const exportBody = JSON.parse(exportRes.body || '{}');
    const jobId = exportBody.id || exportBody.jobId;

    if (jobId) {
      // 6. Poll export job status
      const pollRes = http.get(`${BASE_URL}/api/exports/${jobId}`);
      check(pollRes, {
        'export status retrieved 200': (r) => r.status === 200,
        'job status is valid string': (r) => {
          const job = JSON.parse(r.body || '{}');
          return job.status === 'queued' || job.status === 'processing' || job.status === 'completed';
        },
      });
    }
  }

  sleep(0.1);
}
