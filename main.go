const UA = [
  // Windows (已更新)
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // Windows 11/10 - Chrome
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // Windows 11/10 - Edge
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0", // Windows 11/10 - Firefox
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 OPR/112.0.0.0", // Windows 11/10 - Opera

  // macOS (已更新)
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // macOS Sonoma (Intel) - Chrome
  "Mozilla/5.0 (Macintosh; Apple M1 Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1.15", // macOS Sonoma (Apple Silicon) - Safari
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:129.0) Gecko/20100101 Firefox/129.0", // macOS Sonoma (Intel) - Firefox
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // macOS Sonoma (Intel) - Edge
  "Mozilla/5.0 (Macintosh; Apple M2 Pro Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // macOS Sonoma (Apple Silicon M2) - Chrome

  // Android (保持最新)
  "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36", // Android 14 (Pixel 8 Pro) - Chrome
  "Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Mobile Safari/537.36", // Android 14 (Samsung Galaxy S24 Ultra) - Chrome
  "Mozilla/5.0 (Linux; Android 13; a_real_phone_lol) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36", // Android 13 - Generic Device - Chrome
  "Mozilla/5.0 (Linux; Android 14; sdk_gphone64_arm64; rv:129.0) Gecko/129.0 Firefox/129.0", // Android 14 (Emulator) - Firefox

  // iOS (保持最新)
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", // iPhone - iOS 18 - Safari
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/128.0.6613.25 Mobile/15E148 Safari/604.1", // iPhone - iOS 18 - Chrome
  "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", // iPad - iOS 18 - Safari
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/128.0 Mobile/15E148", // iPhone - iOS 17.5.1 - Firefox

  // Linux (保持最新)
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // Linux (Ubuntu/Debian) - Chrome
  "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:129.0) Gecko/20100101 Firefox/129.0", // Linux (Ubuntu) - Firefox
  "Mozilla/5.0 (X11; Fedora; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36", // Linux (Fedora) - Chrome
];
const CORS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'POST, GET, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization',
};
//固定token值
const TOKEN = 'sk-2025';
const randUA = () => UA[Math.floor(Math.random() * UA.length)];

const json = (obj, status = 200) => new Response(JSON.stringify(obj), { status, headers: { ...CORS, 'Content-Type': 'application/json' } });
const err = (msg, type = 'invalid_request_error', status = 400) => json({ error: { message: msg, type } }, status);
const authErr = () => err('Invalid token', 'authentication_error', 401);

const checkAuth = req => {
  const t = req.headers.get('Authorization')?.slice(7);
  return t === TOKEN ? null : authErr();
};

async function api(url, opts = {}) {
  const ctrl = new AbortController();
  const t = setTimeout(() => ctrl.abort(), 60_000);
  try {
    const res = await fetch(url, { ...opts, signal: ctrl.signal });
    if (!res.ok) throw new Error(`API ${res.status}: ${await res.text()}`);
    return res;
  } finally {
    clearTimeout(t);
  }
}

export default {
  async fetch(req) {
    const { pathname } = new URL(req.url);

    if (req.method === 'OPTIONS') return new Response(null, { headers: CORS, status: 204 });
    if (!['/v1/chat/completions', '/v1/models'].includes(pathname)) return new Response('Not Found', { status: 404 });

    const auth = checkAuth(req);
    if (auth) return auth;

    if (pathname === '/v1/models') {
      if (req.method !== 'GET') return new Response('Method Not Allowed', { status: 405, headers: CORS });
      try {
        const r = await api('https://api.gmi-serving.com/v1/models', {
          headers: { Authorization: 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImFmNDg4M2M0LWQ1NGQtNGMxMC04YjI1LTRhMmVmODZlMDkwOSIsInNjb3BlIjoiaWVfbW9kZWwiLCJjbGllbnRJZCI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCJ9.cmKlYiGKUBPoD6RP9UlJ_cqcCXfFvIsCuIAmsgplebo', 'User-Agent': randUA() },
        });
        return json(await r.json());
      } catch (e) {
        return err(e.message);
      }
    }

    if (req.method !== 'POST') return new Response('Method Not Allowed', { status: 405, headers: CORS });
    try {
      const body = await req.json();
      const stream = body.stream === true;
      const payload = {
        temperature: body.temperature ?? 0.5,
        max_tokens: body.max_tokens ?? 4096,
        top_p: body.top_p ?? 0.95,
        stream,
        messages: body.messages,
        model: body.model || 'Qwen3-Coder-480B-A35B-Instruct-FP8',
      };

      const r = await api('https://console.gmicloud.ai/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': randUA(),
          'Accept': 'application/json, text/plain, */*',
          'Accept-Language': 'zh-CN,zh;q=0.9',
          'Cache-Control': 'no-cache',
          Pragma: 'no-cache',
          Origin: 'https://console.gmicloud.ai',
          Referer: 'https://console.gmicloud.ai/playground/llm/deepseek-v3-0324/e26776ab-2bf8-4a6b-aec0-3307350a8c1e?tab=playground',
          'Sec-Ch-Ua': '"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"',
          'Sec-Ch-Ua-Mobile': '?0',
          'Sec-Ch-Ua-Platform': '"Windows"',
          'Sec-Fetch-Dest': 'empty',
          'Sec-Fetch-Mode': 'cors',
          'Sec-Fetch-Site': 'same-origin',
        },
        body: JSON.stringify(payload),
      });

      if (stream) return new Response(r.body, { headers: { ...CORS, 'Content-Type': 'text/event-stream' } });

      const d = await r.json();
      return json({
        id: `chatcmpl-${Math.random().toString(36).slice(2, 12)}`,
        object: 'chat.completion',
        created: Math.floor(Date.now() / 1000),
        model: body.model || 'Qwen3-Coder-480B-A35B-Instruct-FP8',
        choices: [
          {
            index: 0,
            message: { role: 'assistant', content: d.choices?.[0]?.message?.content || d.result || '' },
            finish_reason: d.choices?.[0]?.finish_reason || 'stop',
          },
        ],
        usage: d.usage || { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
      });
    } catch (e) {
      return err(e.message);
    }
  },
};
