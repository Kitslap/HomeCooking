import { useEffect, useState } from "react"
import type { Page } from "@/App"
import { storage, recipes } from "@/lib/api"
import type { StorageStats, Recipe } from "@/lib/api"

const levelColor: Record<string, string> = { ok: "#5a9e6f", low: "#d4943a", critical: "#c85050" }

export default function Dashboard({ onNavigate }: { onNavigate: (p: Page) => void }) {
  const [stats, setStats]   = useState<StorageStats | null>(null)
  const [recent, setRecent] = useState<Recipe[]>([])
  const [alerts, setAlerts] = useState<any[]>([])

  useEffect(() => {
    storage.stats().then(setStats).catch(() => {})
    recipes.list().then(r => setRecent(r.data?.slice(0, 3) ?? [])).catch(() => {})
    storage.alerts().then(r => setAlerts(r.data?.slice(0, 3) ?? [])).catch(() => {})
  }, [])

  const card = "flex flex-col gap-2 p-4 md:p-5 rounded-xl cursor-pointer transition-all"
  const cardStyle = { background: "#141210", border: "1px solid #2a2018" }
  const cardHover = { background: "#1a1610", border: "1px solid #3a2e22" }

  const StatCard = ({ label, value, icon, page, sub }: any) => (
    <div className={card} style={cardStyle}
      onClick={() => onNavigate(page)}
      onMouseEnter={e => Object.assign((e.currentTarget as HTMLElement).style, cardHover)}
      onMouseLeave={e => Object.assign((e.currentTarget as HTMLElement).style, cardStyle)}>
      <div className="flex items-center justify-between">
        <span className="text-xl" style={{ color: "#d4734a" }}>{icon}</span>
        <span className="text-2xl md:text-3xl font-bold" style={{ color: "#f0e8dc", letterSpacing: "-1px" }}>{value ?? "—"}</span>
      </div>
      <span className="text-xs font-medium uppercase tracking-widest" style={{ color: "#6a5040" }}>{label}</span>
      {sub && <span className="text-xs" style={{ color: "#564a3a" }}>{sub}</span>}
    </div>
  )

  return (
    <div className="space-y-5 max-w-4xl">
      <div>
        <h1 className="text-xl font-semibold" style={{ color: "#f0e8dc" }}>Bonjour 👋</h1>
        <p className="text-sm mt-1" style={{ color: "#6a5040" }}>
          {new Date().toLocaleDateString("fr-FR", { weekday: "long", day: "numeric", month: "long" })}
        </p>
      </div>

      {/* Stat cards : 1 col mobile → 3 cols desktop */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <StatCard label="Stocks" value={stats?.total} icon="▣" page="storage"
          sub={stats ? `${stats.critical_count} critique${stats.critical_count > 1 ? "s" : ""}` : ""} />
        <StatCard label="Alertes" value={stats ? stats.critical_count + stats.low_count : null} icon="◇" page="storage"
          sub={stats?.expiring_count ? `${stats.expiring_count} expire${stats.expiring_count > 1 ? "nt" : ""} bientôt` : "Tout est OK"} />
        <StatCard label="Recettes" value={recent.length > 0 ? recent.length : "0"} icon="◈" page="recipes" sub="Votre bibliothèque" />
      </div>

      {/* Listes : 1 col mobile → 2 cols desktop */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">

        {/* Recettes récentes */}
        <div className="rounded-xl overflow-hidden" style={{ background: "#141210", border: "1px solid #2a2018" }}>
          <div className="flex items-center justify-between px-4 md:px-5 py-3.5" style={{ borderBottom: "1px solid #2a2018" }}>
            <span className="text-xs font-semibold uppercase tracking-widest" style={{ color: "#6a5040" }}>Dernières recettes</span>
            <button onClick={() => onNavigate("recipes")}
              className="text-xs transition-colors" style={{ color: "#564a3a" }}
              onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
              onMouseLeave={e => (e.currentTarget.style.color = "#564a3a")}>
              Voir tout →
            </button>
          </div>
          {recent.length === 0
            ? <p className="px-4 md:px-5 py-4 text-sm" style={{ color: "#564a3a" }}>Aucune recette — cliquez sur + pour commencer !</p>
            : recent.map(r => (
              <div key={r.id} className="px-4 md:px-5 py-3.5 transition-colors" style={{ borderBottom: "1px solid #1e1810" }}
                onMouseEnter={e => (e.currentTarget.style.background = "#1a1610")}
                onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                <div className="text-sm font-medium" style={{ color: "#d8cfc4" }}>{r.name}</div>
                <div className="flex gap-1.5 mt-1.5 flex-wrap">
                  {r.tags?.slice(0, 3).map(t => (
                    <span key={t} className="text-xs px-2 py-0.5 rounded-full"
                      style={{ background: "#221a14", color: "#8a7060", border: "1px solid #2e2418" }}>{t}</span>
                  ))}
                </div>
              </div>
            ))}
        </div>

        {/* Alertes stocks */}
        <div className="rounded-xl overflow-hidden" style={{ background: "#141210", border: "1px solid #2a2018" }}>
          <div className="flex items-center justify-between px-4 md:px-5 py-3.5" style={{ borderBottom: "1px solid #2a2018" }}>
            <span className="text-xs font-semibold uppercase tracking-widest" style={{ color: "#6a5040" }}>Stocks faibles</span>
            <button onClick={() => onNavigate("storage")}
              className="text-xs transition-colors" style={{ color: "#564a3a" }}
              onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
              onMouseLeave={e => (e.currentTarget.style.color = "#564a3a")}>
              Gérer →
            </button>
          </div>
          {alerts.length === 0
            ? <p className="px-4 md:px-5 py-4 text-sm" style={{ color: "#5a9e6f" }}>✓ Tous les stocks sont OK</p>
            : alerts.map(a => (
              <div key={a.id} className="flex items-center justify-between px-4 md:px-5 py-3.5 transition-colors"
                style={{ borderBottom: "1px solid #1e1810" }}
                onMouseEnter={e => (e.currentTarget.style.background = "#1a1610")}
                onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                <div className="flex items-center gap-2.5">
                  <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: levelColor[a.level] ?? "#564a3a" }} />
                  <span className="text-sm" style={{ color: "#d8cfc4" }}>{a.name}</span>
                </div>
                <span className="text-xs" style={{ color: "#8a7060" }}>{a.quantity} {a.unit}</span>
              </div>
            ))}
        </div>
      </div>

      {/* Actions rapides : scroll horizontal sur mobile */}
      <div className="flex gap-3 overflow-x-auto pb-1 -mx-4 px-4 md:mx-0 md:px-0 md:flex-wrap">
        {[
          { label: "Nouvelle recette",    icon: "◈", page: "recipes"  as Page },
          { label: "Mettre à jour stock", icon: "▣", page: "storage"  as Page },
        ].map(a => (
          <button key={a.label} onClick={() => onNavigate(a.page)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm transition-all flex-shrink-0"
            style={{ background: "#141210", border: "1px solid #2a2018", color: "#8a7060" }}
            onMouseEnter={e => { e.currentTarget.style.color = "#d4734a"; e.currentTarget.style.borderColor = "#d4734a40" }}
            onMouseLeave={e => { e.currentTarget.style.color = "#8a7060"; e.currentTarget.style.borderColor = "#2a2018" }}>
            <span>{a.icon}</span> {a.label}
          </button>
        ))}
      </div>
    </div>
  )
}
