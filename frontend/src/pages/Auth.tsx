import { useState } from "react"
import { auth, setToken } from "@/lib/api"
import logoDark from "@/assets/home_cooking_logo_dark.png"

export default function Auth({ onLogin }: { onLogin: () => void }) {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault(); setError(""); setLoading(true)
    try {
      const res = await auth.login(username, password)
      setToken(res.access_token); onLogin()
    } catch (err: any) { setError(err.message || "Erreur") }
    finally { setLoading(false) }
  }

  const inp = "w-full rounded-lg px-4 py-3 text-sm outline-none transition-all"
  const inpStyle = { background: "#1a1410", border: "1px solid #2e2418", color: "#f0e8dc" }
  const inpFocus = { borderColor: "#d4734a" }

  return (
    <div className="flex h-screen items-center justify-center px-4" style={{ background: "#0e0c0b" }}>
      <div className="w-full max-w-sm" style={{ background: "#141210", border: "1px solid #2a2018", borderRadius: "16px", padding: "28px" }}>

        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <img src={logoDark} alt="Home Cooking" className="w-40 mb-3" />
          <div style={{ color: "#6a5040", fontSize: "11px" }}>Connexion à votre espace</div>
        </div>

        <form onSubmit={submit} className="flex flex-col gap-3">
          <div>
            <label className="block text-xs font-medium mb-1.5" style={{ color: "#8a7060" }}>Nom d'utilisateur</label>
            <input value={username} onChange={e => setUsername(e.target.value)}
              type="text" placeholder="admin" required minLength={3} maxLength={32} className={inp}
              style={inpStyle}
              onFocus={e => Object.assign(e.target.style, inpFocus)}
              onBlur={e => Object.assign(e.target.style, { borderColor: "#2e2418" })} />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1.5" style={{ color: "#8a7060" }}>Mot de passe</label>
            <input value={password} onChange={e => setPassword(e.target.value)}
              type="password" placeholder="••••••••••" required minLength={10} className={inp}
              style={inpStyle}
              onFocus={e => Object.assign(e.target.style, inpFocus)}
              onBlur={e => Object.assign(e.target.style, { borderColor: "#2e2418" })} />
          </div>

          {error && (
            <div className="rounded-lg px-3 py-2.5 text-sm" style={{ background: "#c8505018", border: "1px solid #c8505030", color: "#e07070" }}>
              {error}
            </div>
          )}

          <button type="submit" disabled={loading}
            className="mt-2 w-full py-3 rounded-lg text-sm font-semibold transition-all disabled:opacity-50"
            style={{ background: "linear-gradient(135deg, #d4734a, #c05e38)", color: "#fff" }}>
            {loading ? "Chargement…" : "Se connecter"}
          </button>
        </form>

      </div>
    </div>
  )
}
