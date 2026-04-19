const BASE = "/api/v1"

let accessToken: string | null = sessionStorage.getItem("jwt")

export function setToken(t: string | null) {
  accessToken = t
  if (t) sessionStorage.setItem("jwt", t)
  else sessionStorage.removeItem("jwt")
}
export function getToken() { return accessToken }

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" }
  if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`
  const res = await fetch(`${BASE}${path}`, { ...opts, headers })
  if (res.status === 401) { setToken(null); throw new Error("401") }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

// ── Setup (premier lancement) ────────────────────────────────────────────────
export const setupApi = {
  status: () => req<{ needs_setup: boolean }>("/setup/status"),
  create: (username: string, password: string) =>
    req<{ access_token: string; expires_in: number }>("/setup", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
}

export const auth = {
  login:    (username: string, password: string) =>
    req<{ access_token: string }>("/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  register: (username: string, password: string) =>
    req<{ access_token: string }>("/auth/register", { method: "POST", body: JSON.stringify({ username, password }) }),
  logout:   () => req<void>("/auth/logout", { method: "POST" }),
}

export const recipes = {
  list:   (params?: { search?: string; tag?: string; cursor?: string }) => {
    const q = new URLSearchParams(params as Record<string, string>).toString()
    return req<{ data: Recipe[]; next_cursor: string; total: number }>(`/recipes${q ? "?" + q : ""}`)
  },
  get:    (id: number) => req<Recipe>(`/recipes/${id}`),
  create: (body: RecipeInput) => req<Recipe>("/recipes", { method: "POST", body: JSON.stringify(body) }),
  update: (id: number, body: Partial<RecipeInput>) =>
    req<Recipe>(`/recipes/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  delete: (id: number) => req<void>(`/recipes/${id}`, { method: "DELETE" }),
}

export const storage = {
  list:         (params?: { category?: string; level?: string; search?: string }) => {
    const q = new URLSearchParams(params as Record<string, string>).toString()
    return req<{ data: StorageItem[]; total: number }>(`/storage${q ? "?" + q : ""}`)
  },
  stats:        () => req<StorageStats>("/storage/stats"),
  alerts:       () => req<{ data: StorageAlert[]; count: number }>("/storage/alerts"),
  shoppingList: () => req<{ data: ShoppingEntry[]; count: number }>("/storage/shopping-list"),
  get:          (id: number) => req<StorageItem>(`/storage/${id}`),
  create:       (body: StorageInput) => req<StorageItem>("/storage", { method: "POST", body: JSON.stringify(body) }),
  update:       (id: number, body: Partial<StorageInput>) =>
    req<StorageItem>(`/storage/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  adjust:       (id: number, delta: number) =>
    req<StorageItem>(`/storage/${id}/quantity`, { method: "PATCH", body: JSON.stringify({ delta }) }),
  delete:       (id: number) => req<void>(`/storage/${id}`, { method: "DELETE" }),
}

// ── Types ────────────────────────────────────────────────────────────────────
export interface Recipe {
  id: number; user_id: number; name: string; description?: string
  servings: number; prep_time?: number; cook_time?: number
  difficulty?: "facile" | "moyen" | "difficile"
  tags: string[]; image_url?: string
  ingredients?: Ingredient[]; steps?: Step[]
  created_at: string; updated_at: string
}
export interface Ingredient { id: number; name: string; quantity?: number; unit?: string; sort_order: number }
export interface Step { id: number; step_order: number; content: string }
export interface RecipeInput {
  name: string; description?: string; servings: number
  prep_time?: number; cook_time?: number; difficulty?: string
  tags?: string[]; ingredients?: { name: string; quantity?: number; unit?: string }[]
  steps?: { step_order: number; content: string }[]
}
export interface StorageItem {
  id: number; name: string; quantity: number; unit: string
  category?: string; expiry?: string; alert_at: number; notes?: string
  level: "ok" | "low" | "critical"; created_at: string; updated_at: string
}
export interface StorageInput {
  name: string; quantity: number; unit: string
  category?: string; expiry?: string; alert_at?: number; notes?: string
}
export interface StorageStats {
  total: number; ok_count: number; low_count: number
  critical_count: number; expiring_count: number
  attention_count: number // union distincte : low ∪ critical ∪ expiring (articles nécessitant une action)
  categories: string[]
}
export interface StorageAlert {
  id: number; name: string; quantity: number; unit: string
  category?: string; alert_at: number; expiry?: string; level: string
}
export interface ShoppingEntry {
  item_id: number; name: string; need: number; unit: string; category?: string; level: string
}
