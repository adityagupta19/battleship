const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export async function registerUser(username: string): Promise<number> {
  const res = await fetch(`${API_URL}/api/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || 'registration failed')
  }
  const data = await res.json()
  return data.user_id
}
