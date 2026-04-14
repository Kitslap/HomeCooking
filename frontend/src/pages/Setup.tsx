import { useState } from "react"
import { setupApi, setToken } from "@/lib/api"

interface SetupProps {
  onComplete: () => void
}

/** Étapes du wizard */
type Step = 1 | 2 | 3

/** Niveau de robustesse du mot de passe */
type Strength = "none" | "weak" | "medium" | "strong" | "excellent"

export default function Setup({ onComplete }: SetupProps) {
  const [step, setStep] = useState<Step>(1)
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPwd, setShowPwd] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  // ── Validation ──────────────────────────────────────────────────────────
  const usernameRegex = /^[a-zA-Z0-9._-]{3,32}$/
  const isUsernameValid = usernameRegex.test(username)

  const getStrength = (pwd: string): Strength => {
    if (!pwd) return "none"
    let score = 0
    if (pwd.length >= 10) score++
    if (pwd.length >= 14) score++
    if (/[A-Z]/.test(pwd) && /[a-z]/.test(pwd)) score++
    if (/[0-9]/.test(pwd) && /[^A-Za-z0-9]/.test(pwd)) score++
    if (score <= 1) return "weak"
    if (score === 2) return "medium"
    if (score === 3) return "strong"
    return "excellent"
  }

  const strength = getStrength(password)
  const strengthLabels: Record<Strength, string> = {
    none: "", weak: "Faible", medium: "Moyen", strong: "Fort", excellent: "Excellent"
  }
  const strengthColors: Record<Strength, string> = {
    none: "#2a2018", weak: "#c85050", medium: "#c4a040", strong: "#4a9a6a", excellent: "#4a9a6a"
  }
  const strengthBars = strength === "none" ? 0 : strength === "weak" ? 1 : strength === "medium" ? 2 : strength === "strong" ? 3 : 4

  // ── Navigation entre étapes ─────────────────────────────────────────────
  const goStep2 = () => {
    if (!isUsernameValid) {
      setError("Le nom d'utilisateur doit faire 3-32 caractères (lettres, chiffres, ._- uniquement)")
      return
    }
    setError("")
    setStep(2)
  }

  const submit = async () => {
    if (password.length < 10) {
      setError("Le mot de passe doit faire au moins 10 caractères")
      return
    }
    if (password !== confirm) {
      setError("Les mots de passe ne correspondent pas")
      return
    }
    setError("")
    setLoading(true)
    try {
      const res = await setupApi.create(username, password)
      setToken(res.access_token)
      setStep(3)
      // Redirection automatique après 3 secondes
      setTimeout(() => onComplete(), 3000)
    } catch (err: any) {
      setError(err.message || "Erreur lors de la création du compte")
    } finally {
      setLoading(false)
    }
  }

  // ── Styles (cohérents avec la palette existante) ────────────────────────
  const colors = {
    bg: "#0e0c0b",
    card: "#120f0d",
    input: "#1a1410",
    border: "#2a2018",
    borderFocus: "#d4734a",
    accent: "#d4734a",
    accentDark: "#b85a34",
    text: "#f0e8dc",
    textSec: "#8a7060",
    textMuted: "#564a3a",
    error: "#c85050",
    success: "#4a9a6a",
  }

  const inp = "w-full rounded-lg px-4 py-3 text-sm outline-none transition-all"
  const inpStyle = { background: colors.input, border: `1px solid ${colors.border}`, color: colors.text }
  const inpFocus = { borderColor: colors.borderFocus, boxShadow: `0 0 0 3px rgba(212,115,74,0.15)` }

  // ── Dots de progression ─────────────────────────────────────────────────
  const Dots = () => (
    <div className="flex items-center justify-center gap-2 mb-8">
      {[1, 2, 3].map((s, i) => (
        <div key={s} className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full transition-all" style={{
            background: s < step ? colors.success : s === step ? colors.accent : colors.border,
            boxShadow: s === step ? `0 0 8px rgba(212,115,74,0.4)` : "none",
          }} />
          {i < 2 && <div className="w-6 h-0.5 rounded-full transition-all" style={{
            background: s < step ? colors.success : colors.border,
          }} />}
        </div>
      ))}
    </div>
  )

  // ── Eye icon toggle ─────────────────────────────────────────────────────
  const EyeBtn = ({ show, onToggle }: { show: boolean; onToggle: () => void }) => (
    <button type="button" onClick={onToggle}
      className="absolute right-3 top-1/2 -translate-y-1/2 p-1 transition-colors"
      style={{ color: colors.textMuted, background: "none", border: "none", cursor: "pointer" }}>
      {show ? (
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
          <line x1="1" y1="1" x2="23" y2="23"/>
        </svg>
      ) : (
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
        </svg>
      )}
    </button>
  )

  return (
    <div className="flex h-screen items-center justify-center px-4" style={{ background: colors.bg }}>
      <div className="w-full max-w-md">

        {/* Logo */}
        <div className="text-center mb-10">
          <div className="inline-flex w-16 h-16 rounded-2xl items-center justify-center text-2xl font-bold mb-4"
            style={{ background: `linear-gradient(135deg, ${colors.accent}, ${colors.accentDark})`, color: "#fff", boxShadow: "0 8px 32px rgba(212,115,74,0.25)" }}>
            CH
          </div>
          <div className="font-semibold text-lg" style={{ color: colors.text }}>Cooking Home</div>
          <div className="text-xs mt-1" style={{ color: colors.textMuted }}>v0.1 — Configuration initiale</div>
        </div>

        <Dots />

        {/* Card */}
        <div className="rounded-2xl p-8" style={{ background: colors.card, border: `1px solid ${colors.border}`, boxShadow: "0 4px 24px rgba(0,0,0,0.3)" }}>

          {/* ── Étape 1 : Bienvenue + username ────────────────────────── */}
          {step === 1 && (
            <div>
              <h2 className="text-lg font-semibold mb-1" style={{ color: colors.text }}>Bienvenue</h2>
              <p className="text-sm mb-6" style={{ color: colors.textSec, lineHeight: "1.5" }}>
                C'est la première fois que cette instance est lancée. Créons votre compte administrateur pour sécuriser l'accès.
              </p>

              {/* Notice de sécurité */}
              <div className="flex items-start gap-2.5 rounded-lg px-3.5 py-3 mb-6"
                style={{ background: "rgba(212,115,74,0.06)", border: "1px solid rgba(212,115,74,0.15)" }}>
                <svg className="flex-shrink-0 mt-0.5" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={colors.accent} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                </svg>
                <p className="text-xs" style={{ color: colors.textSec, lineHeight: "1.5" }}>
                  Cet assistant n'est disponible qu'au premier lancement. Une fois votre compte créé, cette page ne sera plus accessible.
                </p>
              </div>

              <div className="mb-4">
                <label className="block text-xs font-medium mb-1.5" style={{ color: colors.textSec }}>Nom d'utilisateur</label>
                <input value={username} onChange={e => { setUsername(e.target.value); setError("") }}
                  type="text" placeholder="admin" required minLength={3} maxLength={32} className={inp}
                  style={inpStyle}
                  onFocus={e => Object.assign(e.target.style, inpFocus)}
                  onBlur={e => Object.assign(e.target.style, { borderColor: colors.border, boxShadow: "none" })}
                  onKeyDown={e => e.key === "Enter" && goStep2()}
                  autoFocus />
                <p className="text-xs mt-1" style={{ color: colors.textMuted }}>3-32 caractères : lettres, chiffres, . _ -</p>
              </div>

              {error && (
                <div className="rounded-lg px-3 py-2.5 text-xs mb-4" style={{ background: "#c8505018", border: "1px solid #c8505030", color: "#e07070" }}>
                  {error}
                </div>
              )}

              <button onClick={goStep2}
                className="w-full py-3 rounded-lg text-sm font-semibold transition-all flex items-center justify-center gap-2"
                style={{ background: `linear-gradient(135deg, ${colors.accent}, ${colors.accentDark})`, color: "#fff", boxShadow: "0 2px 12px rgba(212,115,74,0.3)" }}>
                Continuer
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
              </button>
            </div>
          )}

          {/* ── Étape 2 : Mot de passe ────────────────────────────────── */}
          {step === 2 && (
            <div>
              <h2 className="text-lg font-semibold mb-1" style={{ color: colors.text }}>Mot de passe</h2>
              <p className="text-sm mb-6" style={{ color: colors.textSec, lineHeight: "1.5" }}>
                Choisissez un mot de passe robuste. Minimum 10 caractères — privilégiez une phrase de passe.
              </p>

              <div className="mb-4">
                <label className="block text-xs font-medium mb-1.5" style={{ color: colors.textSec }}>Mot de passe</label>
                <div className="relative">
                  <input value={password} onChange={e => { setPassword(e.target.value); setError("") }}
                    type={showPwd ? "text" : "password"} placeholder="••••••••••••" required minLength={10} className={inp}
                    style={inpStyle}
                    onFocus={e => Object.assign(e.target.style, inpFocus)}
                    onBlur={e => Object.assign(e.target.style, { borderColor: colors.border, boxShadow: "none" })}
                    autoFocus />
                  <EyeBtn show={showPwd} onToggle={() => setShowPwd(s => !s)} />
                </div>
                {/* Barre de force */}
                <div className="flex gap-1 mt-2">
                  {[1, 2, 3, 4].map(i => (
                    <div key={i} className="flex-1 h-0.5 rounded-full transition-all" style={{
                      background: i <= strengthBars ? strengthColors[strength] : colors.border,
                    }} />
                  ))}
                </div>
                {strength !== "none" && (
                  <p className="text-xs text-right mt-1 transition-colors" style={{ color: strengthColors[strength] }}>
                    {strengthLabels[strength]}
                  </p>
                )}
              </div>

              <div className="mb-4">
                <label className="block text-xs font-medium mb-1.5" style={{ color: colors.textSec }}>Confirmation</label>
                <div className="relative">
                  <input value={confirm} onChange={e => { setConfirm(e.target.value); setError("") }}
                    type={showConfirm ? "text" : "password"} placeholder="••••••••••••" required className={inp}
                    style={inpStyle}
                    onFocus={e => Object.assign(e.target.style, inpFocus)}
                    onBlur={e => Object.assign(e.target.style, { borderColor: colors.border, boxShadow: "none" })}
                    onKeyDown={e => e.key === "Enter" && submit()} />
                  <EyeBtn show={showConfirm} onToggle={() => setShowConfirm(s => !s)} />
                </div>
              </div>

              {error && (
                <div className="rounded-lg px-3 py-2.5 text-xs mb-4" style={{ background: "#c8505018", border: "1px solid #c8505030", color: "#e07070" }}>
                  {error}
                </div>
              )}

              <div className="flex gap-3 mt-6">
                <button onClick={() => { setStep(1); setError("") }}
                  className="flex-1 py-3 rounded-lg text-sm font-medium transition-all"
                  style={{ background: colors.input, border: `1px solid ${colors.border}`, color: colors.textSec }}>
                  Retour
                </button>
                <button onClick={submit} disabled={loading}
                  className="flex-1 py-3 rounded-lg text-sm font-semibold transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                  style={{ background: `linear-gradient(135deg, ${colors.accent}, ${colors.accentDark})`, color: "#fff", boxShadow: "0 2px 12px rgba(212,115,74,0.3)" }}>
                  {loading ? (
                    <>
                      <div className="w-4 h-4 border-2 rounded-full animate-spin" style={{ borderColor: "rgba(255,255,255,0.3)", borderTopColor: "#fff" }} />
                      Création…
                    </>
                  ) : "Créer le compte"}
                </button>
              </div>
            </div>
          )}

          {/* ── Étape 3 : Succès ──────────────────────────────────────── */}
          {step === 3 && (
            <div className="text-center">
              <div className="inline-flex w-16 h-16 rounded-full items-center justify-center mb-5"
                style={{ background: "rgba(74,154,106,0.12)" }}>
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke={colors.success} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M20 6L9 17l-5-5"/>
                </svg>
              </div>
              <h2 className="text-lg font-semibold mb-2" style={{ color: colors.text }}>Instance configurée</h2>
              <p className="text-sm mb-6" style={{ color: colors.textSec, lineHeight: "1.5" }}>
                Votre compte administrateur a été créé. L'assistant de configuration est maintenant verrouillé définitivement.
              </p>

              {/* Récapitulatif */}
              <div className="rounded-lg p-3.5 mb-6 text-left" style={{ background: colors.input, border: `1px solid ${colors.border}` }}>
                <div className="flex justify-between items-center py-1.5" style={{ borderBottom: `1px solid ${colors.border}`, paddingBottom: "10px", marginBottom: "4px" }}>
                  <span className="text-xs" style={{ color: colors.textMuted }}>Compte</span>
                  <span className="text-sm font-medium" style={{ color: colors.text }}>{username}</span>
                </div>
                <div className="flex justify-between items-center py-1.5" style={{ borderBottom: `1px solid ${colors.border}`, paddingBottom: "10px", marginBottom: "4px" }}>
                  <span className="text-xs" style={{ color: colors.textMuted }}>Rôle</span>
                  <span className="text-sm font-medium" style={{ color: colors.text }}>Administrateur</span>
                </div>
                <div className="flex justify-between items-center py-1.5">
                  <span className="text-xs" style={{ color: colors.textMuted }}>Endpoint /setup</span>
                  <span className="text-sm font-medium" style={{ color: colors.error }}>Verrouillé</span>
                </div>
              </div>

              <button onClick={onComplete}
                className="w-full py-3 rounded-lg text-sm font-semibold transition-all flex items-center justify-center gap-2"
                style={{ background: `linear-gradient(135deg, ${colors.accent}, ${colors.accentDark})`, color: "#fff", boxShadow: "0 2px 12px rgba(212,115,74,0.3)" }}>
                Accéder à l'application
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
              </button>
              <p className="text-xs mt-3" style={{ color: colors.textMuted }}>Redirection automatique dans quelques secondes…</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="text-center text-xs mt-5" style={{ color: colors.textMuted }}>
          Cooking Home — self-hosted
        </p>
      </div>
    </div>
  )
}
