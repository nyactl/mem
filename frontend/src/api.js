const TOKEN_KEY = 'mem_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(t) {
  localStorage.setItem(TOKEN_KEY, t)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

async function req(method, path, body) {
  const headers = { 'Authorization': 'Bearer ' + getToken() }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) {
    clearToken()
    throw new Error('unauthorized')
  }
  if (!res.ok) throw new Error(`${method} ${path}: ${res.status}`)
  return res.status === 204 ? null : res.json()
}

export const api = {
  notes: ()         => req('GET',   '/api/notes'),
  note:  (id)       => req('GET',   `/api/notes/${encodeURIComponent(id)}`),
  create: (payload) => req('POST',  '/api/notes', payload),
  update: (id, payload) => req('PATCH', `/api/notes/${encodeURIComponent(id)}`, payload),
}
